package main

import (
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/pepetka/gocrawler"
)

func writeReport(w io.Writer, report gocrawler.Report) error {
	failedResults := make([]gocrawler.PageResult, 0, len(report.Results))
	redirectedResults := make([]gocrawler.PageResult, 0, len(report.Results))

	if len(report.Results) > 0 {
		tw := tabwriter.NewWriter(w, 0, 8, 2, ' ', 0)

		if _, err := fmt.Fprintln(tw, "DEPTH\tHTTP\tBYTES\tTIME\tCONTENT-TYPE\tURL"); err != nil {
			return err
		}

		for _, result := range report.Results {
			if result.Err != nil {
				failedResults = append(failedResults, result)
			}
			if result.ResponseURL != nil && result.RequestedURL.String() != result.ResponseURL.String() {
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
			if _, err := fmt.Fprintf(tw, "%d\t%s\t%d\t%s\t%s\t%s\n", result.Depth, status, result.BodyBytesRead, result.Duration.Round(time.Millisecond), contentType, result.RequestedURL.Redacted()); err != nil {
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
			if _, err := fmt.Fprintf(w, "%s -> %s\n", redirectedResult.RequestedURL.Redacted(), redirectedResult.ResponseURL.Redacted()); err != nil {
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
			message := strings.ReplaceAll(pageErrorMessage(failedResult.Err), "\n", "\n  ")
			if _, err := fmt.Fprintf(w, "%s: %s\n", failedResult.RequestedURL.Redacted(), message); err != nil {
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

func pageErrorMessage(err error) string {
	requestErr, ok := err.(*url.Error)
	if !ok {
		return err.Error()
	}

	cause := requestErr.Err.Error()
	// net/http quotes the raw Location in its error when the redirect URL cannot be parsed.
	if strings.HasPrefix(cause, "failed to parse Location header ") {
		cause = "invalid redirect Location header"
	}
	target, parseErr := url.Parse(requestErr.URL)
	if parseErr != nil || target.Opaque != "" {
		return fmt.Sprintf("%s: %s", requestErr.Op, cause)
	}
	return fmt.Sprintf("%s %q: %s", requestErr.Op, target.Redacted(), cause)
}
