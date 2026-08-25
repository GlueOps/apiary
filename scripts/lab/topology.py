"""Print which nodes each lab cluster currently holds.

Reads /cluster/resources, which is what the control plane sees — deliberately
not `pvecm status` over SSH. A node can be a corosync member while the API still
reports the old shape for a moment after a join, and it is the API view that
placement acts on, so the API view is the one worth showing.

Usage: topology.py <label> <api-base> <token-id> <secret>

The secret is passed as an argument rather than read from the environment so the
caller decides where it comes from; it is never printed, and a failure prints the
status code rather than the response body, which can echo the token back.
"""

import json
import ssl
import sys
import urllib.error
import urllib.request


def main() -> int:
    if len(sys.argv) != 5:
        print(__doc__.strip().splitlines()[-1], file=sys.stderr)
        return 2
    label, base, token_id, secret = sys.argv[1:5]

    # The lab ships a self-signed certificate, as every un-fronted Proxmox
    # install does. Mirrors Hive.InsecureSkipVerify, and this script only ever
    # reads a node list.
    ctx = ssl.create_default_context()
    ctx.check_hostname = False
    ctx.verify_mode = ssl.CERT_NONE

    req = urllib.request.Request(
        base.rstrip("/") + "/cluster/resources",
        headers={"Authorization": f"PVEAPIToken={token_id}={secret}"},
    )
    try:
        with urllib.request.urlopen(req, timeout=15, context=ctx) as r:
            data = json.load(r).get("data", [])
    except urllib.error.HTTPError as e:
        hint = " (token has privilege separation on?)" if e.code == 403 else ""
        print(f"  {label}: HTTP {e.code}{hint}")
        return 1
    except Exception as e:  # noqa: BLE001 - any failure is "cannot see it"
        print(f"  {label}: unreachable ({type(e).__name__})")
        return 1

    nodes = sorted(r["node"] for r in data if r.get("type") == "node")
    guests = sum(1 for r in data if r.get("type") in ("qemu", "lxc"))
    print(f"  {label}: nodes={nodes} guests={guests}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
