package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func serve() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		timeValue := ""
		now = time.Now()
		tc, err := calculateTime()
		if err != nil {
			fmt.Fprintf(w, "Error loading time")
			return
		}
		if tc.Payperiod.Remaining > 0 {
			timeValue = now.Add(tc.Payperiod.Remaining).Format("15:04")
		} else {
			timeValue = fmt.Sprintf("+%s", -1*tc.Payperiod.Remaining.Round(time.Minute))
		}

		fmt.Fprintf(w, `<html>
<title>Time</title>
<meta name="viewport" content="width=device-width,initial-scale=1">
<style>
body {
display: flex;
width: 100%%;
height: 100%%;
margin: auto;
align-items:center;
justify-content:center;
text-align: center;
}
</style>
<body><h1>%s</h1>`, timeValue)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}
	log.Info("Ready to serve",
		log.Field("port", port),
	)
	http.ListenAndServe(":"+port, nil)
}
