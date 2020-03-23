+++
slug = "gorilla-time-series-database"
date = 2019-11-21
visibility = "published"
+++

# Gorilla Time Series Database

[Gorilla] is an in-memory, time series database from Facebook optimized for writes, reading data in a few milliseconds, and high availability. Facebook open-sourced the code as [Beringei], but the repo is archived. At its core, Gorilla is a 26-hour write-through cache backed by durable storage in [HBase], a distributed key-value store. Gorilla’s contributions include a novel, streaming timestamp compression scheme. Gorilla optimizes for four attributes:

[gorilla]: gorilla.pdf
[hbase]: https://hbase.apache.org/
[beringei]: https://github.com/facebookarchive/beringei

1. High data insertion rate. The primary requirement is that Gorilla should always be available to take writes. The expected insertion rate is 10M data points per second. Each data point consists of a 64 bit timestamp with a 64 bit float value.

2. Real-time monitoring to show new data within tens of seconds.

3. Reads in under one millisecond and fast scans over all in-memory data in tens of milliseconds.

4. Reliability requirements. Gorilla always serves reads even if a server crashes or when an entire region fails.

Gorilla compromises on the following attributes:

- Flexibility. The only supported data type is a named stream of 64 bit floating point values. Higher level abstractions can be built on top of Gorilla.
- Duration. Gorilla only stores the last 26 hours of data.
- Granularity. The minimum granularity is 15 seconds.
- Durability. A server crash can cause data loss of up to 64kB which is 1-2 seconds of data. During prolonged outages, Gorilla preserves the most recent 1 minute of data and discards the rest of the data.
- Consistency. Data is streamed between datacenter regions without attempting to guarantee consistency.
- Query flexibility. Gorilla serves raw compressed blocks to clients. There’s no query engine in Gorilla so clients are expected to process the compressed blocks.

## Time stamp compression

Gorilla introduces a novel lossless compression scheme for streaming timestamps. Gorilla’s timestamp encoding is based on the observation that the vast majority of timestamps arrive at a fixed interval. A fixed interval is common if metrics are pulled from servers at regular intervals, like every 30 seconds. Using sampled production data, the Gorilla team found 96% of timestamps compress to a single bit with the compression scheme. 

To implement the compression, each stream is divided into blocks aligned at two-hour intervals. The block header contains a 64 bit timestamp of the beginning of the block, e.g. `2019-09-05T02:00`. The first timestamp is the delta from the header timestamp stored in 14 bits. Using 14 bits allows one second granularity within the two hour window. Subsequent timestamps are encoded with a double delta scheme as in below:

```
# Block timestamp aligned to 8:00:00.
timestamps =    [8:00:30, 8:01:30, 8:02:30, 8:03:28]
deltas =        [     30,      60,      60,      57]
double_deltas = [       ,      30,       0,      -4]
```

The `double_deltas` are encoded using a variable sized integer encoding similar to the [varint encoding] in Protocol Buffers described below:

[varint encoding]: https://developers.google.com/protocol-buffers/docs/encoding#varints

The `delta_deltas` are encoded using a variable sized integer encoding.

- If the delta is zero, store a single `0` bit.
- If the delta is in `[-63, 64)` store `0b10` followed by the signed value in 7
  bits.
- If the delta is in `[-255, 256)` store `0b110` followed by the signed value in
  9 bits.
- If the delta is in `[-2047, 2048)` store `0b1110` followed by the signed value
  in 12 bits.
- Otherwise, store `0b1111` followed by the delta in 32 bits.

The 4 timestamps would be encoded as follows:

```bash
Block header: Timestamp at 08:00:30
14 bits: 1st Timestamp delta: 30
9 bits: 0b10 + binary(30)
1 bit: 0
9 bits: 0b10 + binary(-4)
```

## Time series value compression

Gorilla uses xor compression for the floating point values associated with a timestamp. Using xor on similar float values drops the sign, exponent, and first few bits of the mantissa from the value which is advantageous because most values don’t change significantly compared to neighboring values. The Gorilla team found that 59% of values are identical to the previous value and compress to a single bit. Since the encoding is variable length, the entire two hour block must be decoded to access values. This isn’t a problem for time series databases because the value of the data lies in aggregation, not in single points.

# Architecture

![Gorilla architecture](gorilla_architecture.png)

Gorilla runs instances in two redundant regions. Both regions consist of many instances. Each instance contains a number of shards.  Each shard contains many named time series. Time series data in blocks and in the append-only log is persisted on a distributed file system for durability.  Gorilla instances replicate shard data to the corresponding shard in the other region but make no attempt to maintain consistency between shards in each region.

A Paxos-based system called ShardManager assigns shards to nodes. I think each time series is contained by a single shard. It’s unclear if Gorilla mitigates hotspots that might occur for frequent metrics.
The in-memory organization of Gorilla is a two-level map:

1. The time series map is the first level map from a shard index to a time series map. The key list contains the mapping between a time series name and its shard index.
2. The time series map maps a string name to a TimeSeries data structure. Additionally, time series map maintains a vector of the same TimeSeries data structures for efficient scans over the entire dataset.
3. The TimeSeries consists of an open block for incoming writes and two-hour closed blocks for the previous 26 hours.

# Query flow

![Gorilla query flow](gorilla_query_flow.png)

The TimeSeries data structure is a collection of closed blocks containing historical data and a single open block containing the previous two hours of data. Upon receiving a query:

1. The Gorilla node checks the keylist to get the index for the shard containing the named time series.  Gorilla uses the index in the shard map to get the pointer to time series map.
2. Gorilla read locks the time series map.
3. Using the map, Gorilla looks up and copies the pointer to the time series, and then releases the read lock.
4. Gorilla spin locks the time series to avoid mutation while data is copied.
 Finally, Gorilla copies the data as it exists to the outgoing RPC.


# Write flow

![Gorilla write flow](gorilla_write_flow.png)

1. First, Gorilla looks up the shard ID to get the directory containing the append-only log for the data.
2. Gorilla appends the data to a shard-specific log.  The log is not a write ahead log. Gorilla only writes data once it has 64kb of data for the shard which is 1-2 seconds of data. The log must include a 32bit integer ID to identify the named timestamp which increases the data size dramatically compared to the compression scheme within a TimeSeries.
3. Similarly to the query flow, Gorilla uses the shard map and time series map to find the correct time series data structure.  Gorilla appends the data to the open block in the time series data structure using the compression algorithm described above.
4. After two hours, the Gorilla node closes all open blocks and flushes each one to disk with a corresponding checkpoint file. After all TimeSeries for a shard are flushed, the Gorilla node deletes the append-only log for that shard.
