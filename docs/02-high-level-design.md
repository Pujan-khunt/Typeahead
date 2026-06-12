# High Level Design

## Trie Based Approach
We have a partial search query and we need to return the top 10 queries whose prefix matches the partial query. The goto for efficient prefix matching is a prefix tree or Trie. We can create a trie which contains all the search queries and the frequency of the query at the leaf node.

```go
type TrieNode struct {
	children  [63]*TrieNode // 63 = A-Z (26) + a-z (26) + 0-9 (10) + space (1)
	frequency int64         // frequency of search query
}
```

<div align="center" style="margin: 30px 0px;">
    <img src="./trie.png" width="500" />
</div>

The problem with this approach is that tries are great when the data is static, meaning it will query the data in an optimal manner, but when adding a new query would take $O(n)$ time which is not fast enough because our workload is write heavy. Also since we need to get the top 10 values based on the frequency of each potential search query, we would have to iterate over every single query whose prefix matches the partial query, so essentially we are iterating over all queries whose prefix matches the partial query, there is no optimization done by the trie.

<div align="center" style="margin: 30px 0px;">
    <img src="./trie-query.png" width="500" />
</div>

This problem can be solved by precomputing the top 10 results for every prefix of a query.

```go
type TrieNode struct {
	children    [63]*TrieNode // 63 = A-Z (26) + a-z (26) + 0-9 (10) + space (1)
	frequency   int64         // frequency of search query
	suggestions [10]string    // top 10 suggestions based on frequency
}
```

This would lead to much faster reads since the time complexity is now just $O(l)$. The writes would be slightly slow since on every search query, the suggestions need to be recalculated for every prefix.

### Sharding
Now the problem shifts to sharding the trie on multiple database servers. The challenge is to find a sharding key which has high cardinality and leads to even distribution of requests and data.

#### 1. First Character Of Query
Using the first character of the query leads to poor data distribution and low cardinality. Data distribution is poor due to some characters being used as the first character of queries way more frequently than others.

<div align="center" style="margin: 20px 0px;">
    <img src="./sharding-first-character.png" width="500" />
</div>

#### 2. First Three Characters Of Query
Using the first three characters of the query solves the low cardinality problem, but the issue of uneven data distribution persists and becomes worse, since now the distribution between keys like "why" and "xzi" is enormously different.

<div align="center" style="margin: 20px 0px;">
    <img src="./sharding-first-three-characters.png" width="500" />
</div>

### Conclusion
There is no database which supports trie natively, so to implement this approach we would have to implement our own database from scratch which should not be the case for developing a MVP. The cons of a trie based approach overweight the pros. We need to think differently, what are we really doing with a trie?
