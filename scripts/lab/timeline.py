"""Put a clock on `go test -v` output from the fleet cycle.

The test already logs its phases; this only prefixes elapsed time and strips the
file:line noise, which is what makes a demo legible — the two gate refusals are
instant while the depart and the join are the parts that take real seconds, and
that contrast is the point being demonstrated.

Reads stdin, writes stdout. Unbuffered on purpose: a demo that prints nothing for
thirty seconds and then everything at once has lost the plot.
"""

import sys
import time


def main() -> int:
    t0 = time.time()
    failed = False
    for line in sys.stdin:
        s = line.rstrip()
        if "fleet_cycle_lab_test.go:" in s:
            # "    fleet_cycle_lab_test.go:117: [sync] ok  ..." -> "[sync] ok  ..."
            print(f"{time.time() - t0:6.1f}s  {s.split(': ', 1)[-1]}", flush=True)
        elif s.startswith("    ") and s.strip():
            print(f"        {s.strip()}", flush=True)
        elif s.startswith(("--- ", "ok  ", "FAIL")):
            failed = failed or s.startswith("FAIL") or s.startswith("--- FAIL")
            print(f"{time.time() - t0:6.1f}s  {s}", flush=True)
    # Exit non-zero on failure: this sits at the end of a pipe, so without it a
    # failed move would leave `just` reporting success.
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
