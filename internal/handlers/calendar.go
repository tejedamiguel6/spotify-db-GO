package handlers

import (
	"net/http"
	"os"
	"time"

	"example.com/spotifydb/internal/repository"
	"github.com/gin-gonic/gin"
)

// maxCalendarRange caps a single request to roughly one year of days.
const maxCalendarRange = 370 * 24 * time.Hour

// defaultCalendarTZ is the timezone whose calendar days plays are grouped
// into. Override with the CALENDAR_TZ env var or a ?tz= query param.
const defaultCalendarTZ = "America/Denver"

// GetCalendar returns per-day listening totals with each day's most-played
// track, for rendering an album-art calendar. Defaults to the current month.
// GET /calendar?from=YYYY-MM-DD&to=YYYY-MM-DD&tz=America/Denver
func GetCalendar(c *gin.Context) {
	tz := c.Query("tz")
	if tz == "" {
		tz = os.Getenv("CALENDAR_TZ")
	}
	if tz == "" {
		tz = defaultCalendarTZ
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'tz' timezone"})
		return
	}

	from, to, err := parseDateRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now().In(loc)
	if from == nil {
		firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		from = &firstOfMonth
	}
	if to == nil {
		to = &now
	}

	if to.Before(*from) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "'to' date is before 'from' date"})
		return
	}
	if to.Sub(*from) > maxCalendarRange {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date range too large (max ~1 year)"})
		return
	}

	days, err := repository.GetDailyCalendar(*from, *to, loc.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if days == nil {
		days = []repository.CalendarDay{}
	}

	c.JSON(http.StatusOK, gin.H{
		"from": from.Format("2006-01-02"),
		"to":   to.Format("2006-01-02"),
		"tz":   loc.String(),
		"days": days,
	})
}
