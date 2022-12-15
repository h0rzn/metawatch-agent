#!/usr/bin/python3
import requests 
from datetime import datetime
from datetime import timedelta


if __name__ == "__main__":

    td = timedelta(seconds=30)
    now = datetime.now()
    tst = now - td
    params = {
        "from": tst
    }
    print(params["from"])

    r = requests.get("http://localhost:8080/containers/e63d12ed9e74cb2d5994e9e0356aad7588939108fb2a1e5ec729274e07e820bf/metrics", params=params)
    print(r.status_code)
    print(r.content)