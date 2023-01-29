import requests
import json

TOKEN = ""
MONGO_CID = "e63d12ed9e74cb2d5994e9e0356aad7588939108fb2a1e5ec729274e07e820bf"

def login():
    p = {
        "username": "master",
        "password": "master"
    }
    r = requests.post("http://localhost:8080/login", params=p)
    print(r.status_code)
    if r.ok:
        return True, r.json()["token"]
    return False, ""

def metrics(amount):
    h = {
        "Authorization": "Bearer " + TOKEN,
        "Content-Type": "application/json"
    }
    p = {
        # "from": "2022-12-15T13:00:00Z",
        "from": "2023-01-10T18:00:00Z",
        # "to": "2022-12-15T13:30:00Z"
        "to": "2023-01-10T18:05:00Z",
        "amount": amount
    }
    r = requests.get("http://localhost:8080/api/containers/e63d12ed9e74cb2d5994e9e0356aad7588939108fb2a1e5ec729274e07e820bf/metrics", headers=h, params=p)
    print(json.dumps(r.json(), indent=4))



if __name__ == "__main__":
    status, token = login()
    if status:
        TOKEN = token
    else:
        print("auth failed")

    metrics(2)


