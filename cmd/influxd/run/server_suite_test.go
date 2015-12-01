package run_test

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"
)

var tests Tests

// Load all shared tests
func init() {
	tests = make(map[string]Test)

	tests["database_commands"] = Test{
		queries: []*Query{
			&Query{
				name:    "create database should succeed",
				command: `CREATE DATABASE db0`,
				exp:     `{"results":[{}]}`,
				once:    true,
			},
			&Query{
				name:    "create database should error with bad name",
				command: `CREATE DATABASE 0xdb0`,
				exp:     `{"error":"error parsing query: found 0, expected identifier at line 1, char 17"}`,
			},
			&Query{
				name:    "show database should succeed",
				command: `SHOW DATABASES`,
				exp:     `{"results":[{"series":[{"name":"databases","columns":["name"],"values":[["db0"]]}]}]}`,
			},
			&Query{
				name:    "create database should error if it already exists",
				command: `CREATE DATABASE db0`,
				exp:     `{"results":[{"error":"database already exists"}]}`,
			},
			&Query{
				name:    "create database should not error with existing database with IF NOT EXISTS",
				command: `CREATE DATABASE IF NOT EXISTS db0`,
				exp:     `{"results":[{}]}`,
			},
			&Query{
				name:    "create database should create non-existing database with IF NOT EXISTS",
				command: `CREATE DATABASE IF NOT EXISTS db1`,
				exp:     `{"results":[{}]}`,
			},
			&Query{
				name:    "show database should succeed",
				command: `SHOW DATABASES`,
				exp:     `{"results":[{"series":[{"name":"databases","columns":["name"],"values":[["db0"],["db1"]]}]}]}`,
			},
			&Query{
				name:    "drop database db0 should succeed",
				command: `DROP DATABASE db0`,
				exp:     `{"results":[{}]}`,
				once:    true,
			},
			&Query{
				name:    "drop database db1 should succeed",
				command: `DROP DATABASE db1`,
				exp:     `{"results":[{}]}`,
				once:    true,
			},
			&Query{
				name:    "drop database should error if it does not exists",
				command: `DROP DATABASE db1`,
				exp:     `{"results":[{"error":"database not found: db1"}]}`,
			},
			&Query{
				name:    "drop database should not error with non-existing database db1 WITH IF EXISTS",
				command: `DROP DATABASE IF EXISTS db1`,
				exp:     `{"results":[{}]}`,
			},
			&Query{
				name:    "show database should have no results",
				command: `SHOW DATABASES`,
				exp:     `{"results":[{"series":[{"name":"databases","columns":["name"]}]}]}`,
			},
			&Query{
				name:    "drop database should error if it doesn't exist",
				command: `DROP DATABASE db0`,
				exp:     `{"results":[{"error":"database not found: db0"}]}`,
			},
		},
	}

	tests["drop_and_recreate_database"] = Test{
		db:    "db0",
		rp:    "rp0",
		write: fmt.Sprintf(`cpu,host=serverA,region=uswest val=23.2 %d`, mustParseTime(time.RFC3339Nano, "2000-01-01T00:00:00Z").UnixNano()),
		queries: []*Query{
			&Query{
				name:    "Drop database after data write",
				command: `DROP DATABASE db0`,
				exp:     `{"results":[{}]}`,
				once:    true,
			},
			&Query{
				name:    "Recreate database",
				command: `CREATE DATABASE db0`,
				exp:     `{"results":[{}]}`,
				once:    true,
			},
			&Query{
				name:    "Recreate retention policy",
				command: `CREATE RETENTION POLICY rp0 ON db0 DURATION 365d REPLICATION 1 DEFAULT`,
				exp:     `{"results":[{}]}`,
				once:    true,
			},
			&Query{
				name:    "Show measurements after recreate",
				command: `SHOW MEASUREMENTS`,
				exp:     `{"results":[{}]}`,
				params:  url.Values{"db": []string{"db0"}},
			},
			&Query{
				name:    "Query data after recreate",
				command: `SELECT * FROM cpu`,
				exp:     `{"results":[{}]}`,
				params:  url.Values{"db": []string{"db0"}},
			},
		},
	}

	tests["drop_database_isolated"] = Test{
		db:    "db0",
		rp:    "rp0",
		write: fmt.Sprintf(`cpu,host=serverA,region=uswest val=23.2 %d`, mustParseTime(time.RFC3339Nano, "2000-01-01T00:00:00Z").UnixNano()),
		queries: []*Query{
			&Query{
				name:    "Query data from 1st database",
				command: `SELECT * FROM cpu`,
				exp:     `{"results":[{"series":[{"name":"cpu","columns":["time","host","region","val"],"values":[["2000-01-01T00:00:00Z","serverA","uswest",23.2]]}]}]}`,
				params:  url.Values{"db": []string{"db0"}},
			},
			&Query{
				name:    "Query data from 1st database with GROUP BY *",
				command: `SELECT * FROM cpu GROUP BY *`,
				exp:     `{"results":[{"series":[{"name":"cpu","tags":{"host":"serverA","region":"uswest"},"columns":["time","val"],"values":[["2000-01-01T00:00:00Z",23.2]]}]}]}`,
				params:  url.Values{"db": []string{"db0"}},
			},
			&Query{
				name:    "Drop other database",
				command: `DROP DATABASE db1`,
				once:    true,
				exp:     `{"results":[{}]}`,
			},
			&Query{
				name:    "Query data from 1st database and ensure it's still there",
				command: `SELECT * FROM cpu`,
				exp:     `{"results":[{"series":[{"name":"cpu","columns":["time","host","region","val"],"values":[["2000-01-01T00:00:00Z","serverA","uswest",23.2]]}]}]}`,
				params:  url.Values{"db": []string{"db0"}},
			},
			&Query{
				name:    "Query data from 1st database and ensure it's still there with GROUP BY *",
				command: `SELECT * FROM cpu GROUP BY *`,
				exp:     `{"results":[{"series":[{"name":"cpu","tags":{"host":"serverA","region":"uswest"},"columns":["time","val"],"values":[["2000-01-01T00:00:00Z",23.2]]}]}]}`,
				params:  url.Values{"db": []string{"db0"}},
			},
		},
	}

	tests["drop_and_recreate_series"] = Test{
		db:    "db0",
		rp:    "rp0",
		write: fmt.Sprintf(`cpu,host=serverA,region=uswest val=23.2 %d`, mustParseTime(time.RFC3339Nano, "2000-01-01T00:00:00Z").UnixNano()),
		queries: []*Query{
			&Query{
				name:    "Show series is present",
				command: `SHOW SERIES`,
				exp:     `{"results":[{"series":[{"name":"cpu","columns":["_key","host","region"],"values":[["cpu,host=serverA,region=uswest","serverA","uswest"]]}]}]}`,
				params:  url.Values{"db": []string{"db0"}},
			},
			&Query{
				name:    "Drop series after data write",
				command: `DROP SERIES FROM cpu`,
				exp:     `{"results":[{}]}`,
				params:  url.Values{"db": []string{"db0"}},
				once:    true,
			},
			&Query{
				name:    "Show series is gone",
				command: `SHOW SERIES`,
				exp:     `{"results":[{}]}`,
				params:  url.Values{"db": []string{"db0"}},
			},
		},
	}
	tests["drop_and_recreate_series_retest"] = Test{
		db:    "db0",
		rp:    "rp0",
		write: fmt.Sprintf(`cpu,host=serverA,region=uswest val=23.2 %d`, mustParseTime(time.RFC3339Nano, "2000-01-01T00:00:00Z").UnixNano()),
		queries: []*Query{
			&Query{
				name:    "Show series is present again after re-write",
				command: `SHOW SERIES`,
				exp:     `{"results":[{"series":[{"name":"cpu","columns":["_key","host","region"],"values":[["cpu,host=serverA,region=uswest","serverA","uswest"]]}]}]}`,
				params:  url.Values{"db": []string{"db0"}},
			},
		},
	}

	tests["drop_series_from_regex"] = Test{
		db: "db0",
		rp: "rp0",
		write: strings.Join([]string{
			fmt.Sprintf(`a,host=serverA,region=uswest val=23.2 %d`, mustParseTime(time.RFC3339Nano, "2000-01-01T00:00:00Z").UnixNano()),
			fmt.Sprintf(`aa,host=serverA,region=uswest val=23.2 %d`, mustParseTime(time.RFC3339Nano, "2000-01-01T00:00:00Z").UnixNano()),
			fmt.Sprintf(`b,host=serverA,region=uswest val=23.2 %d`, mustParseTime(time.RFC3339Nano, "2000-01-01T00:00:00Z").UnixNano()),
			fmt.Sprintf(`c,host=serverA,region=uswest val=30.2 %d`, mustParseTime(time.RFC3339Nano, "2000-01-01T00:00:00Z").UnixNano()),
		}, "\n"),
		queries: []*Query{
			&Query{
				name:    "Show series is present",
				command: `SHOW SERIES`,
				exp:     `{"results":[{"series":[{"name":"a","columns":["_key","host","region"],"values":[["a,host=serverA,region=uswest","serverA","uswest"]]},{"name":"aa","columns":["_key","host","region"],"values":[["aa,host=serverA,region=uswest","serverA","uswest"]]},{"name":"b","columns":["_key","host","region"],"values":[["b,host=serverA,region=uswest","serverA","uswest"]]},{"name":"c","columns":["_key","host","region"],"values":[["c,host=serverA,region=uswest","serverA","uswest"]]}]}]}`,
				params:  url.Values{"db": []string{"db0"}},
			},
			&Query{
				name:    "Drop series after data write",
				command: `DROP SERIES FROM /a.*/`,
				exp:     `{"results":[{}]}`,
				params:  url.Values{"db": []string{"db0"}},
				once:    true,
			},
			&Query{
				name:    "Show series is gone",
				command: `SHOW SERIES`,
				exp:     `{"results":[{"series":[{"name":"b","columns":["_key","host","region"],"values":[["b,host=serverA,region=uswest","serverA","uswest"]]},{"name":"c","columns":["_key","host","region"],"values":[["c,host=serverA,region=uswest","serverA","uswest"]]}]}]}`,
				params:  url.Values{"db": []string{"db0"}},
			},
			&Query{
				name:    "Drop series from regex that matches no measurements",
				command: `DROP SERIES FROM /a.*/`,
				exp:     `{"results":[{}]}`,
				params:  url.Values{"db": []string{"db0"}},
				once:    true,
			},
			&Query{
				name:    "make sure DROP SERIES doesn't delete anything when regex doesn't match",
				command: `SHOW SERIES`,
				exp:     `{"results":[{"series":[{"name":"b","columns":["_key","host","region"],"values":[["b,host=serverA,region=uswest","serverA","uswest"]]},{"name":"c","columns":["_key","host","region"],"values":[["c,host=serverA,region=uswest","serverA","uswest"]]}]}]}`,
				params:  url.Values{"db": []string{"db0"}},
			},
			&Query{
				name:    "Drop series with WHERE field should error",
				command: `DROP SERIES FROM c WHERE val > 50.0`,
				exp:     `{"results":[{"error":"DROP SERIES doesn't support fields in WHERE clause"}]}`,
				params:  url.Values{"db": []string{"db0"}},
			},
			&Query{
				name:    "make sure DROP SERIES with field in WHERE didn't delete data",
				command: `SHOW SERIES`,
				exp:     `{"results":[{"series":[{"name":"b","columns":["_key","host","region"],"values":[["b,host=serverA,region=uswest","serverA","uswest"]]},{"name":"c","columns":["_key","host","region"],"values":[["c,host=serverA,region=uswest","serverA","uswest"]]}]}]}`,
				params:  url.Values{"db": []string{"db0"}},
			},
			&Query{
				name:    "Drop series with WHERE time should error",
				command: `DROP SERIES FROM c WHERE time > now() - 1d`,
				exp:     `{"results":[{"error":"DROP SERIES doesn't support time in WHERE clause"}]}`,
				params:  url.Values{"db": []string{"db0"}},
			},
		},
	}

	tests["retention_policy_commands"] = Test{
		db: "db0",
		queries: []*Query{
			&Query{
				name:    "create retention policy should succeed",
				command: `CREATE RETENTION POLICY rp0 ON db0 DURATION 1h REPLICATION 1`,
				exp:     `{"results":[{}]}`,
				once:    true,
			},
			&Query{
				name:    "create retention policy should error if it already exists",
				command: `CREATE RETENTION POLICY rp0 ON db0 DURATION 1h REPLICATION 1`,
				exp:     `{"results":[{"error":"retention policy already exists"}]}`,
			},
			&Query{
				name:    "show retention policy should succeed",
				command: `SHOW RETENTION POLICIES ON db0`,
				exp:     `{"results":[{"series":[{"columns":["name","duration","replicaN","default"],"values":[["rp0","1h0m0s",1,false]]}]}]}`,
			},
			&Query{
				name:    "alter retention policy should succeed",
				command: `ALTER RETENTION POLICY rp0 ON db0 DURATION 2h REPLICATION 3 DEFAULT`,
				exp:     `{"results":[{}]}`,
				once:    true,
			},
			&Query{
				name:    "show retention policy should have new altered information",
				command: `SHOW RETENTION POLICIES ON db0`,
				exp:     `{"results":[{"series":[{"columns":["name","duration","replicaN","default"],"values":[["rp0","2h0m0s",3,true]]}]}]}`,
			},
			&Query{
				name:    "dropping default retention policy should not succeed",
				command: `DROP RETENTION POLICY rp0 ON db0`,
				exp:     `{"results":[{"error":"retention policy is default"}]}`,
			},
			&Query{
				name:    "show retention policy should still show policy",
				command: `SHOW RETENTION POLICIES ON db0`,
				exp:     `{"results":[{"series":[{"columns":["name","duration","replicaN","default"],"values":[["rp0","2h0m0s",3,true]]}]}]}`,
			},
			&Query{
				name:    "create a second non-default retention policy",
				command: `CREATE RETENTION POLICY rp2 ON db0 DURATION 1h REPLICATION 1`,
				exp:     `{"results":[{}]}`,
				once:    true,
			},
			&Query{
				name:    "show retention policy should show both",
				command: `SHOW RETENTION POLICIES ON db0`,
				exp:     `{"results":[{"series":[{"columns":["name","duration","replicaN","default"],"values":[["rp0","2h0m0s",3,true],["rp2","1h0m0s",1,false]]}]}]}`,
			},
			&Query{
				name:    "dropping non-default retention policy succeed",
				command: `DROP RETENTION POLICY rp2 ON db0`,
				exp:     `{"results":[{}]}`,
				once:    true,
			},
			&Query{
				name:    "show retention policy should show just default",
				command: `SHOW RETENTION POLICIES ON db0`,
				exp:     `{"results":[{"series":[{"columns":["name","duration","replicaN","default"],"values":[["rp0","2h0m0s",3,true]]}]}]}`,
			},
			&Query{
				name:    "Ensure retention policy with unacceptable retention cannot be created",
				command: `CREATE RETENTION POLICY rp3 ON db0 DURATION 1s REPLICATION 1`,
				exp:     `{"results":[{"error":"retention policy duration must be at least 1h0m0s"}]}`,
				once:    true,
			},
			&Query{
				name:    "Check error when deleting retention policy on non-existent database",
				command: `DROP RETENTION POLICY rp1 ON mydatabase`,
				exp:     `{"results":[{"error":"database not found: mydatabase"}]}`,
			},
		},
	}

}

func (tests Tests) load(t *testing.T, key string) Test {
	test, ok := tests[key]
	if !ok {
		t.Fatalf("no test %q", key)
	}

	return test.duplicate()
}
