package main

import (
	"fmt"
	"net/http"
	"time"
)

func serve() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		timeValue := ""
		tc, err := calculateTime()
		if err != nil {
			fmt.Fprintf(w, "Error loading time")
			return
		}
		if tc.Payperiod.Remaining > 0 {
			timeValue = now.Add(tc.Payperiod.Remaining).Format("15:04")
		} else {
			timeValue = "+" + string(-1*tc.Payperiod.Remaining.Round(time.Minute))
		}

		fmt.Fprintf(w, `<html>
		<title>Time</title>
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

	http.ListenAndServe(":8080", nil)
}
