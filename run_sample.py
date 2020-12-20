import random
from collections import defaultdict

import requests

uri = "http://localhost:8080"
rides = defaultdict(dict)
customers = defaultdict(dict)


def add_some_rides():
    payloads = [
        {"name": "Roller Coster", "capacity": "4", "ride_time_secs": "210"},
        {"name": "Bumper car", "capacity": "10", "ride_time_secs": "600"},
        {"name": "Cable car", "capacity": "2", "ride_time_secs": "120"},
        {"name": "Viking", "capacity": "20", "ride_time_secs": "180"},
    ]

    for payload in payloads:
        response = requests.request("POST", uri + "/ride/add", data=payload)


def get_rides():
    response = requests.request("GET", uri + "/ride")
    response_rides = response.json()
    for r in response_rides:
        rides[r["id"]] = r
        print(
            "Ride: {name}, Capacity: {capacity}, Ride time (ns): {ride_time}, Waiting time (ns): {waiting_time}, Queue count: {queue_count}".format(
                name=r["name"],
                waiting_time=r["waiting_time_in_ns"],
                queue_count=r["in_queue_count"],
                capacity=r["capacity"],
                ride_time=r["ride_time_in_ns"],
            )
        )


def add_some_customers(nos: int):
    for i in range(1, nos):
        response = requests.request("POST", uri + "/customer/enter")


def get_customers():
    response = requests.request("GET", uri + "/customer")
    response_customers = response.json()
    for r in response_customers:
        customers[r["id"]] = r


def queue_random_customers_at_random_rides(nos: int):
    for i in range(1, nos):
        customer_id = random.choice(list(customers.keys()))
        ride_id = random.choice(list(rides.keys()))
        payload = {"id": customer_id, "ride_id": ride_id}
        response = requests.request("POST", uri + "/customer/queue", data=payload)


def main():
    # Add required data - needed only the first time
    # add_some_rides()
    # add_some_customers(1000)

    # Queue random customers to rides
    # get_customers()
    # get_rides()
    # queue_random_customers_at_random_rides(100)

    get_rides()


if __name__ == "__main__":
    main()
