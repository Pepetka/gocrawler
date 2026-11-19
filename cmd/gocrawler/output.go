package main

import (
	"fmt"
	"io"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/pepetka/gocrawler"
)

func writeReport(w io.Writer, report gocrawler.Report) error {
	failedResults := make([]gocrawler.Result, 0, len(report.Results))
	redirectedResults := make([]gocrawler.Result, 0, len(report.Results))

	if len(report.Results) > 0 {
		tw := tabwriter.NewWriter(w, 0, 8, 2, ' ', 0)

		if _, err := fmt.Fprintln(tw, "DEPTH\tHTTP\tBYTES\tTIME\tCONTENT-TYPE\tURL"); err != nil {
			return err
		}

		for _, result := range report.Results {
			if result.Err != nil {
				failedResults = append(failedResults, result)
			}
			if result.FinalURL != nil && result.URL.String() != result.FinalURL.String() {
				redirectedResults = append(redirectedResults, result)
			}
			status := strconv.Itoa(result.StatusCode)
			if result.StatusCode == 0 {
				status = "-"
			}
			contentType := result.ContentType
			if result.ContentType == "" {
				contentType = "-"
			}
			if _, err := fmt.Fprintf(tw, "%d\t%s\t%d\t%s\t%s\t%s\n", result.Depth, status, result.BytesRead, result.Duration.Round(time.Millisecond), contentType, result.URL); err != nil {
				return err
			}
		}

		if err := tw.Flush(); err != nil {
			return err
		}

		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	if len(redirectedResults) > 0 {
		if _, err := fmt.Fprintln(w, "Redirects:"); err != nil {
			return err
		}

		for _, redirectedResult := range redirectedResults {
			if _, err := fmt.Fprintf(w, "%s -> %s\n", redirectedResult.URL, redirectedResult.FinalURL); err != nil {
				return err
			}
		}

		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	if len(failedResults) > 0 {
		if _, err := fmt.Fprintln(w, "Errors:"); err != nil {
			return err
		}

		for _, failedResult := range failedResults {
			if _, err := fmt.Fprintf(w, "%s: %s\n", failedResult.URL, failedResult.Err); err != nil {
				return err
			}
		}

		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "Started: %d Succeeded: %d Failed: %d\nDuration: %s\nStop reason: %s", report.Started, report.Succeeded, report.Failed, report.Duration.Round(time.Millisecond), report.StopReason); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	return nil
}
