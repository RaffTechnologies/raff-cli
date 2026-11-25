package commands

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/rafftechnologies/raff-go"
)

const upgradeTestID = "8f2c1d6e-3b4a-4c5d-9e8f-0a1b2c3d4e5f"

// fakeUpgradeAPI answers GET of the database with getBody and the upgrade
// POST with postStatus and postBody, and counts the upgrade calls.
type fakeUpgradeAPI struct {
	getBody    string
	postStatus int
	postBody   string
	posts      int
}

func (f *fakeUpgradeAPI) start(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/databases/"+upgradeTestID:
			_, _ = w.Write([]byte(f.getBody))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/"+upgradeTestID+"/upgrade":
			f.posts++
			w.WriteHeader(f.postStatus)
			_, _ = w.Write([]byte(f.postBody))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	// An empty home keeps any saved config out of the test.
	t.Setenv("HOME", t.TempDir())
	oldURL, oldKey, oldOut := flagAPIURL, flagAPIKey, flagOutput
	flagAPIURL, flagAPIKey, flagOutput = srv.URL, "test", "table"
	t.Cleanup(func() { flagAPIURL, flagAPIKey, flagOutput = oldURL, oldKey, oldOut })
}

func upgradeTestDB(status, engineVersion, extra string) string {
	return `{"success":true,"database":{"database_id":"` + upgradeTestID + `","name":"orders","engine":"postgres",` +
		`"engine_version":"` + engineVersion + `","status":"` + status + `"` + extra + `}}`
}

func upgradeTestRecord(state, message string) string {
	return `,"last_version_upgrade":{"from_version":"18","to_version":"19","state":"` + state + `","message":"` + message +
		`","started_at":"2026-11-24T10:00:00Z","finished_at":""}`
}

// runUpgrade runs 'raff database upgrade' with stdin as the answer to the
// prompt and returns what it printed and its error.
func runUpgrade(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	inR, inW, _ := os.Pipe()
	_, _ = io.WriteString(inW, stdin)
	_ = inW.Close()
	outR, outW, _ := os.Pipe()
	oldIn, oldOut := os.Stdin, os.Stdout
	os.Stdin, os.Stdout = inR, outW
	cmd := newDBUpgradeCmd()
	cmd.SetArgs(args)
	cmd.SilenceUsage, cmd.SilenceErrors = true, true
	err := cmd.Execute()
	os.Stdin, os.Stdout = oldIn, oldOut
	_ = outW.Close()
	out, _ := io.ReadAll(outR)
	return string(out), err
}

func TestUpgradeWithoutVersionListsTheChoices(t *testing.T) {
	cases := []struct {
		name, extra, want string
	}{
		{"versions", `,"upgrade_versions":["19"]`, "--version is required. This database can be upgraded to: 19. Run: raff database upgrade " + upgradeTestID + " --version 19"},
		{"note", `,"upgrade_versions":[],"upgrade_note":"PostgreSQL 19 needs the vector extension updated first."`, "--version is required. PostgreSQL 19 needs the vector extension updated first."},
		{"newest", `,"upgrade_versions":[]`, "--version is required. This database is on the newest version."},
	}
	for _, c := range cases {
		api := &fakeUpgradeAPI{getBody: upgradeTestDB("running", "18", c.extra)}
		api.start(t)
		_, err := runUpgrade(t, "", upgradeTestID)
		if err == nil || err.Error() != c.want {
			t.Errorf("%s: error %v, want %q", c.name, err, c.want)
		}
		if api.posts != 0 {
			t.Errorf("%s: upgraded without --version", c.name)
		}
	}
}

func TestUpgradeDeclinedDoesNothing(t *testing.T) {
	api := &fakeUpgradeAPI{}
	api.start(t)
	out, err := runUpgrade(t, "n\n", upgradeTestID, "--version", "19")
	if err != nil || !strings.Contains(out, "Aborted.") || api.posts != 0 {
		t.Errorf("err %v, posts %d, output %q", err, api.posts, out)
	}
}

func TestUpgradeStartsWithoutWait(t *testing.T) {
	api := &fakeUpgradeAPI{postStatus: http.StatusOK, postBody: upgradeTestDB("upgrading", "18", upgradeTestRecord("backing_up", ""))}
	api.start(t)
	out, err := runUpgrade(t, "y\n", upgradeTestID, "--version", "19")
	want := "Upgrade of " + upgradeTestID + " to PostgreSQL 19 started. Follow it with 'raff database get " + upgradeTestID + "'."
	if err != nil || !strings.Contains(out, want) || api.posts != 1 {
		t.Errorf("err %v, posts %d, output %q", err, api.posts, out)
	}
}

func TestUpgradeWaitSucceeds(t *testing.T) {
	api := &fakeUpgradeAPI{
		postStatus: http.StatusOK,
		postBody:   upgradeTestDB("upgrading", "18", upgradeTestRecord("backing_up", "")),
		getBody:    upgradeTestDB("running", "19", upgradeTestRecord("analyzing", "")),
	}
	api.start(t)
	out, err := runUpgrade(t, "", upgradeTestID, "--version", "19", "--wait", "--force")
	if err != nil || !strings.Contains(out, "orders now runs PostgreSQL 19.") {
		t.Errorf("err %v, output %q", err, out)
	}
}

func TestUpgradeWaitFailedPrintsTheReason(t *testing.T) {
	const reason = "The upgrade to PostgreSQL 19 could not be completed, so your database is back on PostgreSQL 18, unchanged."
	api := &fakeUpgradeAPI{
		postStatus: http.StatusOK,
		postBody:   upgradeTestDB("upgrading", "18", upgradeTestRecord("backing_up", "")),
		getBody:    upgradeTestDB("running", "18", upgradeTestRecord("failed", reason)),
	}
	api.start(t)
	_, err := runUpgrade(t, "", upgradeTestID, "--version", "19", "--wait", "--force")
	if err == nil || err.Error() != reason {
		t.Errorf("error %v, want %q", err, reason)
	}
}

func TestUpgradeRefusalKeepsTheAPIMessage(t *testing.T) {
	const refusal = "Column public.events.payload uses type abstime, which PostgreSQL 19 cannot carry over."
	api := &fakeUpgradeAPI{postStatus: http.StatusPreconditionFailed, postBody: `{"success":false,"error":"` + refusal + `"}`}
	api.start(t)
	_, err := runUpgrade(t, "", upgradeTestID, "--version", "19", "--force")
	if err == nil || !strings.Contains(err.Error(), refusal) {
		t.Errorf("error %v, want the API message", err)
	}
}

// get shows the versions a database can take, else the note on why it has
// none, else nothing.
func TestDetailShowsUpgradeVersionsOrNote(t *testing.T) {
	const note = "PostgreSQL 19 needs the vector extension updated first."
	versions, none := []string{"19"}, []string{}
	notePtr, empty := note, ""
	cases := []struct {
		name     string
		versions *[]string
		note     *string
		want     string
		notWant  []string
	}{
		{"versions", &versions, &notePtr, "Can upgrade to:", []string{"Upgrade:"}},
		{"note", &none, &notePtr, "Upgrade:", []string{"Can upgrade to:"}},
		{"neither", &none, &empty, "", []string{"Upgrade:", "Can upgrade to:"}},
	}
	for _, c := range cases {
		outR, outW, _ := os.Pipe()
		oldOut := os.Stdout
		os.Stdout = outW
		printDatabaseDetail(&raff.Database{UpgradeVersions: c.versions, UpgradeNote: c.note})
		os.Stdout = oldOut
		_ = outW.Close()
		b, _ := io.ReadAll(outR)
		out := string(b)
		if c.want != "" && !strings.Contains(out, c.want) {
			t.Errorf("%s: missing %q in %q", c.name, c.want, out)
		}
		if c.name == "note" && !strings.Contains(out, note) {
			t.Errorf("note: the API's note is missing in %q", out)
		}
		for _, nw := range c.notWant {
			if strings.Contains(out, nw) {
				t.Errorf("%s: unexpected %q in %q", c.name, nw, out)
			}
		}
	}
}

func TestDescribeUpgrade(t *testing.T) {
	state := func(s raff.DatabaseVersionUpgradeState) *raff.DatabaseVersionUpgradeState { return &s }
	from, to, at := "18", "19", "2026-11-24T10:01:00Z"
	msg := "back on PostgreSQL 18, unchanged."
	cases := []struct {
		u    raff.DatabaseVersionUpgrade
		want string
	}{
		{raff.DatabaseVersionUpgrade{FromVersion: &from, ToVersion: &to, State: state(raff.DatabaseUpgradeDone), FinishedAt: &at}, "Upgraded from 18 to 19 on " + at},
		{raff.DatabaseVersionUpgrade{FromVersion: &from, ToVersion: &to, State: state(raff.DatabaseUpgradeFailed), FinishedAt: &at, Message: &msg}, "Upgrade from 18 to 19 on " + at + " did not go through: " + msg},
		{raff.DatabaseVersionUpgrade{FromVersion: &from, ToVersion: &to, State: state(raff.DatabaseUpgradeBackingUp), StartedAt: &at}, "Upgrading from 18 to 19 (backing up, started " + at + ")"},
	}
	for _, c := range cases {
		if got := describeUpgrade(&c.u); got != c.want {
			t.Errorf("got %q, want %q", got, c.want)
		}
	}
}
