package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"sort"
	"strconv"
	"time"

	"github.com/blang/semver"
	"github.com/brimstone/logger"
	"github.com/go-yaml/yaml"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
)

type Config struct {
	APIToken       string
	Projects       map[int64]Project
	Clients        map[int64]Client
	SkipProjects   []string
	RenameProjects map[string]string
}

var (
	config  Config
	log     = logger.New()
	now     = time.Now()
	version = "0.0.0"
)

func get(url string, v interface{}) error {
	url = "https://api.track.toggl.com/api/v8/" + url
	log.Debug("debug",
		log.Field("url", url),
	)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(config.APIToken, "api_token")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(v)
}

func getCurrent() (TimeEntry, error) {
	var current TimeEntriesCurrentResponse
	err := get("time_entries/current", &current)
	if err != nil {
		return current.Data, fmt.Errorf("error getting current time entries %w", err)
	}
	if current.Data.Duration == 0 {
		return TimeEntry{}, nil
	}
	if _, ok := config.Projects[current.Data.Pid]; !ok {
		err = loadProject(current.Data.Pid)
		if err != nil {
			return current.Data, fmt.Errorf("error getting missing project: %w", err)
		}
	}
	return current.Data, nil
}

func loadConfig() error {
	user, err := user.Current()
	if err != nil {
		return err
	}
	configFile := user.HomeDir + "/.togglstat.yaml"
	if _, err = os.Stat(configFile); os.IsNotExist(err) {
		config.Clients = make(map[int64]Client)
		config.Projects = make(map[int64]Project)
		return nil
	}

	c, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(c, &config)
	if err != nil {
		return err
	}
	return nil
}

func saveConfig() error {
	user, err := user.Current()
	if err != nil {
		return err
	}
	f, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(user.HomeDir+"/.togglstat.yaml", f, 0777)
	return err
}

func loadClient(cid int64) error {
	var c ClientResponse
	err := get("clients/"+strconv.FormatInt(cid, 10), &c)
	if err != nil {
		return fmt.Errorf("error getting client %d: %w", cid, err)
	}
	if c.Data.Name == "" {
		return fmt.Errorf("unable to load client name for %d:%w", cid, err)
	}
	config.Clients[cid] = c.Data
	saveConfig()
	return nil
}
func loadProject(pid int64) error {
	var p ProjectResponse
	err := get("projects/"+strconv.FormatInt(pid, 10), &p)
	if err != nil {
		return fmt.Errorf("error loading projects: %w", err)
	}
	if _, ok := config.Clients[p.Data.Cid]; !ok {
		err = loadClient(p.Data.Cid)
		if err != nil {
			return fmt.Errorf("error getting related client: %w", err)
		}
	}
	config.Projects[pid] = p.Data
	saveConfig()
	return nil
}

func getPayperiod(now time.Time) (time.Time, time.Time, error) {
	var (
		payperiodStart time.Time
		payperiodEnd   time.Time
		err            error
	)
	if now.Day() < 16 {
		payperiodStart, err = time.ParseInLocation("2006 January 02",
			fmt.Sprintf("%d %s 01", now.Year(), now.Month()),
			now.Location())
		payperiodEnd, err = time.ParseInLocation("2006 January 02",
			fmt.Sprintf("%d %s 16", now.Year(), now.Month()),
			now.Location())
	} else {
		payperiodStart, err = time.ParseInLocation("2006 January 02",
			fmt.Sprintf("%d %s 16", now.Year(), now.Month()),
			now.Location())
		end := now.AddDate(0, 0, 17)
		payperiodEnd, err = time.ParseInLocation("2006 January 02",
			fmt.Sprintf("%d %s 16", end.Year(), end.Month()),
			now.Location())
	}
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("Error calculating payperiod border: %w", err)
	}
	//payperiodEnd = payperiodStart.Add(time.Hour * 24 * 15)
	return payperiodStart, payperiodEnd, nil
}

func getEntriesSince(start time.Time, end time.Time) ([]TimeEntry, error) {
	var response TimeEntriesResponse
	startFormat := url.QueryEscape(start.Format("2006-01-02T15:04:05-07:00"))
	endFormat := url.QueryEscape(end.Format("2006-01-02T15:04:05-07:00"))
	err := get("time_entries?start_date="+startFormat+"&end_date="+endFormat, &response)
	for _, e := range response {
		if e.Pid == 0 { // TODO handle this a bit better, ignore or delete the entry?
			return nil, fmt.Errorf("Entry is missing a project %s", e.Start.In(now.Location()))
		}
		if _, ok := config.Projects[e.Pid]; !ok {
			err = loadProject(e.Pid)
			if err != nil {
				return nil, fmt.Errorf("error getting missing project: %w", err)
			}
		}
		log.Debug("Entry",
			log.Field("Start", e.Start),
			log.Field("Duration", e.Duration),
			log.Field("Project", config.Projects[e.Pid].Name),
		)
	}
	return response, err
}

func formatDuration(d time.Duration) (ret string) {
	d1 := d.Truncate(time.Hour)
	d -= d1
	ret += fmt.Sprintf("%02.0f:", d1.Hours())
	ret += fmt.Sprintf("%02.0f", d.Minutes())
	return
}

func colorRange(low float64, f string, i float64) string {
	if i < low {
		return "\x1b[1;31m" + f + "\x1b[0m"
	}
	return f
}

func skipProject(project string) bool {
	for _, p := range config.SkipProjects {
		if p == project {
			return true
		}
	}
	return false
}

func main() {
	err := loadConfig()
	if err != nil {
		panic(err)
	}

	// Handle any command line flags
	flag.StringVar(&config.APIToken, "token", config.APIToken, "API Token for Toggl")
	flag.Parse()

	switch flag.Arg(0) {
	case "upgrade":
		fmt.Println("Checking and applying upgrade")
		v := semver.MustParse(version)
		latest, err := selfupdate.UpdateSelf(v, "brimstone/togglstat")
		if err != nil {
			log.Println("Binary update failed:", err)
			os.Exit(1)
		}
		if latest.Version.Equals(v) {
			// latest version is the same as current version. It means current binary is up to date.
			log.Println("Current binary is the latest version", version)
		} else {
			log.Println("Successfully updated to version", latest.Version)
			log.Println("Release note:\n", latest.ReleaseNotes)
		}
		return
	case "version":
		fmt.Printf("togglstat version %s\n", version)
		return
	}

	if config.APIToken == "" {
		// save the config so the user can update it there if they'd like instead
		saveConfig()
		panic(fmt.Errorf("must have an apitoken"))
	}

	payperiodStart, payperiodEnd, err := getPayperiod(now)

	log.Debug("Now",
		log.Field("now", now),
		log.Field("payperiodStart", payperiodStart),
		log.Field("payperiodEnd", payperiodEnd),
	)

	entries, err := getEntriesSince(payperiodStart, payperiodEnd)
	if err != nil {
		panic(err)
	}

	payperiodDays := payperiodEnd.Sub(payperiodStart) / (time.Hour * 24)
	var (
		payperiodDuration time.Duration
		today             time.Duration
		payperiodTarget   time.Duration
		days              = make([]map[string]time.Duration, payperiodDays)
	)
	for i := 0; i < int(payperiodEnd.Sub(payperiodStart)/(time.Hour*24)); i++ {
		log.Debug("Day",
			log.Field("i", i),
		)
		// TODO had to add an extra hour here during the payperiod with the fall daylight savings
		day := payperiodStart.AddDate(0, 0, i)
		if now.Before(day) {
			break
		}
		wd := day.Weekday()
		if wd != time.Saturday && wd != time.Sunday {
			log.Debug("Weekday?",
				log.Field("i", i),
				log.Field("wd", wd),
			)
			payperiodTarget += time.Hour * 8
		}

		log.Debug("Entries",
			log.Field("len", len(entries)),
		)
		for _, e := range entries {
			s := e.Start.In(now.Location())
			if s.Year() != day.Year() || s.Month() != day.Month() || s.Day() != day.Day() {
				log.Debug("Entry is outside current year, month, day",
					log.Field("s", fmt.Sprintf("%d-%d-%d", s.Year(), s.Month(), s.Day())),
					log.Field("day", fmt.Sprintf("%d-%d-%d", day.Year(), day.Month(), day.Day())),
				)
				continue
			}

			if skipProject(config.Clients[config.Projects[e.Pid].Cid].Name) {
				continue
			}
			d := time.Duration(e.Duration) * time.Second
			if d < 0 {
				d = now.Sub(e.At)
			}
			payperiodDuration += d
			if s.Year() == now.Year() && s.Month() == now.Month() && s.Day() == now.Day() {
				today += d
			}
			if days[i] == nil {
				days[i] = make(map[string]time.Duration)
			}
			project := config.Projects[e.Pid].Name
			if np, ok := config.RenameProjects[project]; ok {
				project = np
			}
			log.Debug("Adding Entry",
				log.Field("day", i),
				log.Field("project", project),
				log.Field("duration", d),
			)
			days[i][project] += d
		}
	}

	// TODO
	// if time is before ETA to 8 hours: show ETA in red
	// (not useful?) if time is before ETA to average to payperiod: show ETA in yellow
	// if time is before ETA to payperiod: show ETA in green
	// if time is after ETA to payperiod: show relative time in black

	remaining := time.Hour*8 - today
	payperiodRemaining := payperiodTarget - payperiodDuration

	if remaining > 0 {
		fmt.Printf("\x1b[1;31m%s\x1b[0m", now.Add(remaining).Format("15:04"))
	} else {
		fmt.Printf("+%s", -1*remaining.Round(time.Minute))
	}
	if payperiodRemaining > 0 {
		fmt.Printf("-\x1b[1;31m%s\x1b[0m\n", now.Add(payperiodRemaining).Format("15:04"))
	} else {
		fmt.Printf("+%s\n", -1*payperiodRemaining.Round(time.Minute))
	}

	fmt.Printf("---\n")
	fmt.Printf("Day Duration: %s\n", formatDuration(today))
	fmt.Printf("Day Remaining: %s\n", formatDuration(remaining))
	if remaining > 0 {
		fmt.Printf("Day ETA: %s\n", now.Add(remaining).Format("15:04"))
	} else {
		fmt.Printf("Day Overage: +%s\n", -1*remaining.Round(time.Minute))
	}
	fmt.Printf("Payperiod Target: %s\n", payperiodTarget)
	fmt.Printf("Payperiod Duration: %s\n", formatDuration(payperiodDuration))
	fmt.Printf("Payperiod Remaining: %s\n", formatDuration(payperiodRemaining))
	if payperiodRemaining > 0 {
		fmt.Printf("Payperiod ETA: %s\n", now.Add(payperiodTarget-payperiodDuration).Format("15:04"))
	} else {
		fmt.Printf("Payperiod Overage: +%s\n", -1*payperiodRemaining.Round(time.Minute))
	}

	// Figure all of the projects into sorted slice
	projectsMap := make(map[string]bool)
	for _, d := range days {
		for p := range d {
			projectsMap[p] = true
		}
	}
	var projects []string
	var projectsNameLen int64 = 5
	for p := range projectsMap {
		projects = append(projects, p)
		if int64(len(p)) > projectsNameLen {
			projectsNameLen = int64(len(p))
		}
	}
	sort.Strings(projects)

	// Display header row
	fmt.Printf("%-"+strconv.FormatInt(projectsNameLen, 10)+"s", "")
	for i := range days {
		day := payperiodStart.AddDate(0, 0, i)
		if day.Weekday() == time.Sunday || day.Weekday() == time.Saturday {
			fmt.Printf(" \x1b[3m%s\x1b[0m", day.Format("Jan 02"))
		} else {
			fmt.Printf(" %s", day.Format("Jan 02"))
		}
	}

	suffix := ""
	if _, ok := os.LookupEnv("BitBarDarkMode"); ok {
		suffix = " | font-Menlo trim=false"
	}
	fmt.Println(" Total" + suffix)
	// Print each project per day
	for _, p := range projects {
		fmt.Printf("%-"+strconv.FormatInt(projectsNameLen, 10)+"s", p)
		var projectTotal float64
		for i, d := range days {
			day := payperiodStart.AddDate(0, 0, i)
			r := d[p].Round(time.Minute * 15).Hours()
			w := strconv.FormatInt(int64(len(day.Format("Jan 02"))), 10)
			if r > 0 {
				fmt.Printf(" %"+w+".2f", r)
			} else {
				fmt.Printf(" %"+w+"s", "")
			}
			projectTotal += r
		}
		fmt.Printf(" %5.2f%s\n", projectTotal, suffix)
	}
	// Print footer, the totals per day per project
	fmt.Printf("%-"+strconv.FormatInt(projectsNameLen, 10)+"s", "Total")
	var payperiodTotal float64
	for i, d := range days {
		var projectTotal float64
		day := payperiodStart.AddDate(0, 0, i)
		for _, p := range projects {
			r := d[p].Round(time.Minute * 15).Hours()
			projectTotal += r
			payperiodTotal += r
		}
		w := strconv.FormatInt(int64(len(day.Format("Jan 02"))), 10)
		if projectTotal > 0 {
			fmt.Printf(colorRange(8, " %"+w+".2f", projectTotal), projectTotal)
		} else {
			fmt.Printf(" %"+w+"s", "")
		}
	}
	fmt.Printf("\x1b[1m %5.2f\x1b[0m%s\n", payperiodTotal, suffix)
}
