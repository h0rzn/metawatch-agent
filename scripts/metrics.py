#!/usr/bin/python3
import requests
import json

if __name__ == "__main__":
    params = {
        # "from": "2022-12-15T13:00:00Z",
        "from": "2023-01-10T18:50:00Z",
        # "to": "2022-12-15T13:30:00Z"
        "to": "2023-01-10T18:55:00Z"
    }

    r = requests.get("http://localhost:8080/containers/e63d12ed9e74cb2d5994e9e0356aad7588939108fb2a1e5ec729274e07e820bf/metrics", params=params)
    print(r.status_code)
    print(json.dumps(r.json(), indent=4))