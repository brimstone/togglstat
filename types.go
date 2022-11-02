package main

import "time"

type TimeEntry struct {
	At       time.Time `json:"at"`
	Billable bool      `json:"billable"`
	Duration int64     `json:"duration"`
	Duronly  bool      `json:"duronly"`
	ID       int64     `json:"id"`
	Pid      int64     `json:"pid"`
	Start    time.Time `json:"start"`
	UID      int64     `json:"uid"`
	Wid      int64     `json:"wid"`
}

type TimeEntriesCurrentResponse struct {
	Data TimeEntry `json:"data"`
}

type Project struct {
	Active        bool   `json:"active"`
	ActualHours   int64  `json:"actual_hours"`
	At            string `json:"at"`
	AutoEstimates bool   `json:"auto_estimates"`
	Billable      bool   `json:"billable"`
	Cid           int64  `json:"cid"`
	Color         string `json:"color"`
	CreatedAt     string `json:"created_at"`
	HexColor      string `json:"hex_color"`
	ID            int64  `json:"id"`
	IsPrivate     bool   `json:"is_private"`
	Name          string `json:"name"`
	Template      bool   `json:"template"`
	Wid           int64  `json:"wid"`
}

type ProjectResponse struct {
	Data Project `json:"data"`
}

type Client struct {
	At   string `json:"at"`
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Wid  int64  `json:"wid"`
}

type ClientResponse struct {
	Data Client `json:"data"`
}

type TimeEntriesResponse []TimeEntry
