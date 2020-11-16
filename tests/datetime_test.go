package go_ora_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	_ "github.com/sijms/go-ora"
)

func exec(t *testing.T, s string) {
	_, err := DB.Exec(s)
	if err != nil {
		t.Fatal(err)
	}

}

func dropTable(t *testing.T, tbl string) {
	_, err := DB.Exec("drop table " + tbl)
	if err != nil && !strings.HasPrefix(err.Error(), "ORA-00942") {
		t.Fatal(err)
	}
}

// Oracle DATETIME literals https://docs.oracle.com/cd/B19306_01/server.102/b14200/sql_elements003.htm#BABGIGCJ

// queryTime return the simplest literal for a given datetime
func queryTime(t time.Time) string {
	location := t.Location().String()
	h, mi, s := t.Clock()
	ns := t.Nanosecond()

	if location == "Local" && ns == 0 {
		if h == 0 && mi == 0 && s == 0 {
			return "DATE '" + t.Format("2006-01-02") + "'"
		}
		return "TO_DATE ('" + t.Format("2006-01-02 15:04:05") + "','YYYY-MM-DD HH24:MI:SS')"
	}
	if location == "Local" {
		return "TIMESTAMP '" + t.Format("2006-01-02 15:04:05.000000") + "'"
	}
	return "TIMESTAMP '" + t.Format("2006-01-02 15:04:05.000000 -07:00") + "'"
}

var Shanghai, _ = time.LoadLocation("Asia/Shanghai")
var LosAngeles, _ = time.LoadLocation("America/Los_Angeles")

// getTestDateValues return tests values based on given local time
func getTestDateValues(local *time.Location) []time.Time {
	return []time.Time{
		time.Date(2020, 12, 31, 0, 0, 0, 0, local),
		time.Date(2020, 12, 31, 15, 16, 17, 0, Shanghai),
		time.Date(2020, 12, 31, 15, 16, 17, 0, LosAngeles),
		time.Date(2020, 11, 31, 15, 16, 17, 1.23467e8, local),
		time.Date(2020, 11, 31, 15, 16, 17, 1.23467e8, time.UTC),
		time.Date(2020, 11, 31, 15, 16, 17, 1.23467e8, Shanghai),
		time.Date(2020, 11, 31, 15, 16, 17, 1.23467e8, LosAngeles),
	}
}

func TestSelectTZ(t *testing.T) {
	exec(t, "alter session set time_zone= '7:00'")
	dropTable(t, "TEST_TZ")
	exec(t, `CREATE TABLE TEST_TZ 
			(
				ID NUMBER 
				, D DATE
				, TS TIMESTAMP 
				, TSTZ TIMESTAMP WITH TIME ZONE
				, TSLTZ TIMESTAMP WITH LOCAL TIME ZONE
				, S NCHAR(200) 
			)`)

	for i, tt := range getTestDateValues(Shanghai) {
		t.Run(queryTime(tt), func(t *testing.T) {
			// Insert the test value
			qT := queryTime(tt)
			query := fmt.Sprintf("insert into TEST_TZ (ID,D,TS,TSTZ, TSLTZ,S) values (%d, %s, %s, %s, %s, '%s')", i, qT, qT, qT, qT, strings.Replace(qT, "'", "''", -1))
			stmt, err := DB.Prepare(query)
			if err != nil {
				t.Fatalf("Can't prepare query: %s", err)
				return
			}
			defer stmt.Close()
			_, err = stmt.Exec()
			if err != nil {
				t.Fatalf("Can't exec query:\n%q\n   %s", query, err)
				return
			}

			// Test inserted values
			query = "select D,TS,TSTZ,TSLTZ,S from TEST_TZ where ID = :1"
			stmt, err = DB.Prepare(query)

			if err != nil {
				t.Errorf("Can't prepare query: %s", err)
				return
			}
			rows, err := stmt.Query(i)
			if err != nil {
				t.Errorf("Can't exec query:\n%q\n   %s", query, err)
				return
			}
			defer rows.Close()

			if !rows.Next() {
				t.Errorf("Can't select for ID %d: %s", i, err)
				return
			}

			var (
				d, ts, tsz, tslz time.Time
				s                string
			)

			err = rows.Scan(&d, &ts, &tsz, &tslz, &s)
			if err != nil {
				t.Errorf("Query can't scan row: %s", err)
				return
			}

			if !d.Equal(tt) {
				t.Errorf("Expecting DATE to be %s, got %s, diff:%s (%s)", tt, d, tt.Sub(d), qT)
				return

			}

		})
	}
}

func ExtractDate(t time.Time, loc *time.Location) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

func EqualTime(got, want time.Time) bool {
	d := want.Sub(got)
	if d > 0 {
		return d < 10*time.Minute
	}
	return -d < 10*time.Minute
}

func TestCurrentTime(t *testing.T) {

	// dbLocation := time.UTC
	// sessionLocation := time.Local

	var testValues = []struct {
		s string
		// f func(got, excepted time.Time) bool
	}{
		// https://oracle-base.com/articles/misc/oracle-dates-timestamps-and-intervals#date
		{"SYSDATE"},      // Returns the current date-time from the operating system of the database server.
		{"CURRENT_DATE"}, // Returns the current date-time within the sessions time zone.

		// https://oracle-base.com/articles/misc/oracle-dates-timestamps-and-intervals#timestamp
		{"SYSTIMESTAMP"},      // Returns the current TIMESTAMP from the operating system of the database server to the specified precision. If no precision is specified the default is 6.
		{"CURRENT_TIMESTAMP"}, // Similar to the SYSTIMESTAMP function, but returns the current TIMESTAMP WITH TIME ZONE within the sessions time zone to the specified precision. If no precision is specified the default is 6
		{"LOCALTIMESTAMP"},    // Similar to the current_timestamp function, but returns the current TIMESTAMP with time zone within the sessions time zone to the specified precision. If no precision is specified the default is 6.
	}

	for _, tt := range testValues {
		t.Run(tt.s, func(t *testing.T) {
			q := "select " + tt.s + " from dual"
			stmt, err := DB.Prepare(q)
			if err != nil {
				t.Fatalf("Can't prepare %q: %s", q, err)
				return
			}
			defer stmt.Close()

			rows, err := stmt.Query()
			if err != nil {
				t.Fatalf("Can't query %q: %s", q, err)
				return
			}
			defer rows.Close()

			if !rows.Next() {
				t.Fatalf("Query %q return now rows", q)

			}

			var got time.Time

			err = rows.Scan(&got)
			if err != nil {
				t.Fatalf("Can't scan %q: %s", q, err)
				return
			}

			t.Errorf("%s returns %s", tt.s, got)

		})

	}
}
