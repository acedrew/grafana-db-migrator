package sqlite

import (
	"fmt"
	"io/ioutil"
	"regexp"
)

// Sanitize cleans up a SQLite dump file to prep it for import into Postgres.
func Sanitize(dumpFile string) error {
	data, err := ioutil.ReadFile(dumpFile)
	if err != nil {
		return err
	}

	// Remove SQLite-specific PRAGMA statements
	// and statements that start with BEGIN
	// and statements pertaining to the sqlite_sequence table.
	re := regexp.MustCompile(`(?m)[\r\n]?^(PRAGMA.*;|BEGIN.*;|.*sqlite_sequence.*;)$`)
	sanitized := re.ReplaceAll(data, nil)

	// Ensure there are quotes around table names to avoid using reserved table names like user.
	// This also converts backticks around table names to double quotes.
	re = regexp.MustCompile(`(?msU)^(INSERT INTO) ["` + "`" + `]?([a-zA-Z0-9_]*)["` + "`" + `]? (VALUES.*;)$`)
	sanitized = re.ReplaceAll(sanitized, []byte(`$1 "$2" $3`))

	return ioutil.WriteFile(dumpFile, sanitized, 0644)
}

// CustomSanitize allows you to expand upon the default Sanitize function
// by providing your own regex matcher and replacement to modify data from the dump file.
func CustomSanitize(dumpFile string, regex string, replacement []byte) error {
	re := regexp.MustCompile(regex)
	data, err := ioutil.ReadFile(dumpFile)
	if err != nil {
		return err
	}

	sanitized := re.ReplaceAll(data, replacement)

	return ioutil.WriteFile(dumpFile, sanitized, 0644)

}


// RemoveCreateStatements takes all the CREATE statements out of a dump
// so that no new tables are created.
func RemoveCreateStatements(dumpFile string) error {
	re := regexp.MustCompile(`(?msU)[\r\n]+^CREATE.*;$`)
	data, err := ioutil.ReadFile(dumpFile)
	if err != nil {
		return err
	}
	sanitized := re.ReplaceAll(data, nil)
	return ioutil.WriteFile(dumpFile, sanitized, 0644)
}

// HexDecode takes a file path containing a SQLite dump and
// decodes any hex-encoded data it finds.
func HexDecode(dumpFile string) error {
	re := regexp.MustCompile(`(?m)X\'([a-fA-F0-9]+)\'`)
	data, err := ioutil.ReadFile(dumpFile)
	if err != nil {
		return err
	}

	// Define a function to wrap encoded hex data in a call to decode hexstring.
	decodeHex := func(hexEncoded []byte) []byte {
		return []byte(fmt.Sprintf("'%s%s'", `\x`, re.FindSubmatch(hexEncoded)[1]))
	}

	// Replace regex matches from the dumpFile using the `decodeHex` function defined above.
	sanitized := re.ReplaceAllFunc(data, decodeHex)
	return ioutil.WriteFile(dumpFile, sanitized, 0644)
}

// RemoveAlertRulePausedColumn removes the is_paused column from alert_rule and alert_rule_version INSERT statements
// SQLite may have this column but Grafana 9.2.20 PostgreSQL schema doesn't include it yet
func RemoveAlertRulePausedColumn(dumpFile string) error {
	data, err := ioutil.ReadFile(dumpFile)
	if err != nil {
		return err
	}

	// Match alert_rule INSERT statements and remove the last column (is_paused)
	// The pattern matches everything from INSERT to the semicolon
	// (?s) makes . match newlines in case statements span multiple lines
	re := regexp.MustCompile(`(?s)INSERT INTO "alert_rule" VALUES\(.*?\);`)

	sanitized := re.ReplaceAllFunc(data, func(match []byte) []byte {
		// Replace the last occurrence of ,<digit>); with );
		// The is_paused column is always the last value before );
		lastValuePattern := regexp.MustCompile(`,\d+\);$`)
		result := lastValuePattern.ReplaceAll(match, []byte(");"))
		return result
	})

	// Also fix alert_rule_version table
	re = regexp.MustCompile(`(?s)INSERT INTO "alert_rule_version" VALUES\(.*?\);`)
	sanitized = re.ReplaceAllFunc(sanitized, func(match []byte) []byte {
		lastValuePattern := regexp.MustCompile(`,\d+\);$`)
		result := lastValuePattern.ReplaceAll(match, []byte(");"))
		return result
	})

	return ioutil.WriteFile(dumpFile, sanitized, 0644)
}
