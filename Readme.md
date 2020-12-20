## Main goal
To estimate waiting times for each ride which is updated whenever a customer enters the queue or exits

## DB design
- Database using postgres with gorm as access layer to support other DB's. Tests run using in-memory SQLite DB.
- Apart from the basic `Customer` and `Ride` models to store meta info the rest are all Event sourced

## Events sourcing
- Events Table stores raw and immutable logs of activity in the system. Used for tracking customer movements and ride status.
    - **source_id**: ID based on the event aggregate. For eg. in customer aggregates it will be customerID.
    - **at**: Timestamp of the event used to sort events and play inorder of happening
    - **aggregate_root**: Defines what type of event this belongs to, eg. customer / ride
    - **name**: event name, used to decode the raw data into right place
    - **data**: raw data in whatever format the event wants the data to be stored. In this case we store JSON
    - **ends_at**: Since events are immutable, this can be used for Tombstoning when we want to reduce no of events to process on every call. Another way of doing this would be to use snaphot events
- To get latest of a customer's state we can fetch all events for customer id filtered by `customer` aggregate_root sorted by at. And we can play all these events to get the latest state. Each event defines what changes it does to state.

### Customer events
- Defines the logs of customer activity of either queuing for a ride or leaving the queue.
- [CustomerQueued](events/customers/events.go#L18) defines when a customer joins a queue for a ride. It holds start time and end time, end is calculated based on the ride's current estimated wait time. Customer will be auto removed when queue event end time runs out.
- [CustomerUnQueued](events/customers/events.go#L62) defines when a customer leaves a queue before completing the ride.

### Ride events
- Defines the logs of ride queue activity
- [RideCustomerQueued](events/rides/events.go#L18) defines when a customer joins the ride queue. After adding the customer we re-calcualte the waiting time based on the no of people in queue, capacity & ride time. When a batch finishes the ride the ride time is re-calculated by doing a re-aggregate of the events.
- [RideCustomerUnQueued](events/rides/events.go#L57) defines when a customer leaves the ride queue. After adding the customer we re-calcualte the waiting time based on the no of people in queue, capacity & ride time.

## Cache
- A simple in-memory caching is used in order to reduce no of DB calls and re-processing of raw events.
- After each aggregation of events we cache the result either with a TTL or not based on the state.
- On successfully adding new log entries we will also invalidate the cache for that sourceID.

## Tests
- Api tests are in `api/`
- Event processing tests are in `events/**`

## Examples/usage
- To run the app all you need is docker. And you can run it using `docker-compose up`
- Once running you can use the following to test it
    - [run_sample.py](run_sample.py)
    - [Postman export](UniversalStudios.postman_collection.json)


# Notes

Trying to answer the questions from requirements.

1. How does your application server receive ride current capacity, information, etc?

    Using then endpoint `/ride/add`

1. How do we know the amount of people in a queue of a ride?

    `/ride/` endpoint returns all the rides with it's current waiting time & no of people in queue.

1.  How do we calculate the estimated wait-time for a ride? And how does that propagate to all customers?

    Waiting times are calculated by playing the immutable events for rides. When a customer queue's for the ride we add a log saying so, then on getting the current state we process all ride's events and calculate how many are on queue as of now and how long would it take to clear that.

1. At peak time, the data traffic can grow as much as 100 times of regular times.

    This should be fine as long as we have a dristributed cache like redis across multiple serving apps. Also we can implement snapshotting events to reduce the DB load.

1. Detecting if a ride is malfunctioning or is no longer running.

    This is where the event sourcing works great. We can create a new type of event for ride_inactive or something and get it into play by defining it's Aggregate.
