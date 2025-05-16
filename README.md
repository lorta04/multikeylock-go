# multikeylock

A lightweight Go library for implementing **multi-key mutexes** using only `sync.Map`.

## About

This repository was created out of my own interest in solving the problem of race conditions within the same logical domain — such as when a backend server needs to handle many operations under a single user concurrently.

The library demonstrates how to implement a lock system where each key (e.g., a user ID) has its own mutex, and access is coordinated using Go’s `sync.Map` — and nothing else.

Although the idea of using `sync.Map` alone to manage per-key locking is something I came up with a long time ago, this implementation is *vibe-coded* with the help of ChatGPT and not entirely written by hand.

It seems that a well-known, dedicated library for this exact approach does not yet exist in the Go ecosystem — despite it being such a common concurrency pattern. This repository is also intended for my own personal use in future projects.

## Sister Repository

This Go implementation is intended to be accompanied by a **Rust version** of the same component (non-vibe-coded), which will follow a similar locking model but use idiomatic Rust constructs.

## License

MIT
