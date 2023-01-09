#!/usr/bin/python3
import requests
import json

if __name__ == "__main__":
    params = {
        "from": "2022-12-15T13:00:00Z",
        "to": "2022-12-15T13:30:00Z"
    }

    r = requests.get("http://localhost:8080/containers/61ef6dda76c9aa7e354d7cce951f27d3dff68f2e17c0ca86644ba8885de522d0/metrics", params=params)
    print(r.status_code)
    print(json.dumps(r.json(), indent=4))