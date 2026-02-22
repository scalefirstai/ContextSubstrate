## ADDED Requirements

### Requirement: Content-hash-based cache storage
The system SHALL provide a JSONL-based cache layer stored in `.ctx/cache/entries.jsonl` that maps content hashes to cached artifacts.

#### Scenario: Store and retrieve cache entry
- **WHEN** a CacheEntry is stored via Put and later retrieved via Get with matching content hash and artifact type
- **THEN** the retrieved entry contains all original fields including Payload, TokensIn, and TokensOut

#### Scenario: Cache miss returns nil
- **WHEN** Get is called with a content hash that does not exist in the cache
- **THEN** nil is returned with no error

### Requirement: Cache key deduplication
The system SHALL replace existing cache entries when a new entry with the same key is stored.

#### Scenario: Put replaces existing entry
- **WHEN** Put is called with an entry whose key matches an existing entry
- **THEN** the existing entry is replaced and the total entry count does not increase

### Requirement: Cache invalidation by content hash
The system SHALL provide an Invalidate function that removes all entries matching given content hashes.

#### Scenario: Invalidate removes matching entries
- **WHEN** Invalidate is called with a list of content hashes
- **THEN** all entries with matching ContentHash values are removed and the count of removed entries is returned

#### Scenario: Invalidate with no matches
- **WHEN** Invalidate is called with hashes that don't exist in the cache
- **THEN** zero entries are removed and no error occurs

### Requirement: Cache listing
The system SHALL provide a List function that returns all cache entries.

#### Scenario: List returns all entries
- **WHEN** List is called on a cache with N entries
- **THEN** all N entries are returned

### Requirement: Deterministic cache output
Cache entries SHALL be written in sorted order by key to ensure deterministic, reproducible files.
