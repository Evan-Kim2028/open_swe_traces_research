"""repair_imports rewrites stale module prefixes in hidden tests, their visible copy, the
checksum guards and the patches, and leaves a consistent task alone."""
import hashlib

from openswe_traces.authoring import repair_imports as RI

TEST = 'package http\n\nimport (\n\tgoahttp "example.internal/goa/http"\n)\n'


def make_task(tmp_path, test=TEST, rung=6):
    t = tmp_path / f"httpenc-L{rung}"
    src = t / "environment/src"
    (src / "http").mkdir(parents=True)
    (src / "go.mod").write_text("module example.internal/apikit/v3\n\ngo 1.25\n")
    (t / "tests/hidden/http").mkdir(parents=True)
    (t / "tests/hidden/http/enc_hidden_test.go").write_text(test)
    if rung >= 5:
        (src / "http/enc_hidden_test.go").write_text(test)
    digest = hashlib.sha256(test.encode()).hexdigest()
    (t / "tests/test.sh").write_text(
        f'echo "{digest}  $HIDDEN/http/enc_hidden_test.go" | sha256sum -c\n'
        f'echo "{digest}  /app/http/enc_hidden_test.go" | sha256sum -c\n')
    (t / "tests/gold.patch").write_text(
        '--- a/http/enc.go\n+++ b/http/enc.go\n@@ -1,3 +1,3 @@\n'
        ' \tgoa "example.internal/goa/pkg"\n')
    (t / "instruction.md").write_text("Fix the encoder.\n")
    return t


def test_rewrites_every_copy_and_refreshes_guards(tmp_path):
    t = make_task(tmp_path)
    assert RI.repair(t) == {"example.internal/goa": "example.internal/apikit/v3"}
    new = (t / "tests/hidden/http/enc_hidden_test.go").read_text()
    assert '"example.internal/apikit/v3/http"' in new and "goa/" not in new
    assert (t / "environment/src/http/enc_hidden_test.go").read_text() == new
    digest = hashlib.sha256(new.encode()).hexdigest()
    assert (t / "tests/test.sh").read_text().count(digest) == 2
    assert '"example.internal/apikit/v3/pkg"' in (t / "tests/gold.patch").read_text()
    assert RI.broken_imports(t) == {}


def test_consistent_task_is_untouched(tmp_path):
    t = make_task(tmp_path, TEST.replace("goa/http", "apikit/v3/http"), rung=0)
    (t / "tests/gold.patch").write_text("--- a/x\n+++ b/x\n")
    before = (t / "tests/test.sh").read_text()
    assert RI.repair(t) == {}
    assert (t / "tests/test.sh").read_text() == before
