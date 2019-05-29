---------
Consensus
---------

Consensus is the algorithm by which a set of computers come to
agreement on which possible state is correct, and thus guarantee
one consistent, global view of the state of the system.

Eventual finality
-----------------

All PoW systems use eventual finality, where the resource cost
of creating a block is extremely high. After many blocks are gossiped,
the longest chain of blocks has the most work invested in it,
and thus is the true chain. The "true" head of the chain can switch,
in a process called "chain reorganization". But the probability of such
a reorganization decreases exponentially the more blocks are built
on top of it. Thus, in Bitcoin, the "6 block rule" means that if there
are 6 blocks build on top of the block with your transaction, you can be
extremely confident that no chain reorganization will ever
generate a new true chain that does not include that block. Note this
is not a guarantee that it cannot happen, just that the cost of doing
so becomes so prohibitively high that is very unlikely to ever happen.

Many early PoS systems, such as BitShares, used voting instead of work
to mine blocks, but still used the "longest chain wins" consensus
algorithm. However, this has the critical
`nothing at stake <https://github.com/ethereum/wiki/wiki/Problems#8-proof-of-stake>`__ problem, since the cost of "mining"
blocks on 2, 3, or even 100 alternate chains is quite low.

Another issue here is that any state may have to be reverted, and the
data store must maintain an "undo history" to undo several blocks and
apply others. And clients must wait several blocks (minutes to hours)
before they can take off-chain actions based on the transaction
(eg. give you goods for a blockchain payment).

Immediate finality
------------------

An alternative approach used to guarantee constency comes out of
academic research into Byzantine Fault Tolerance from the 80s and 90s,
which "culminated" in `PBFT <http://pmg.csail.mit.edu/papers/osdi99.pdf>`__ .
`Tendermint <https://tendermint.com/>`__ uses an algorithm very similar
to PBFT with optimizations learned from blockchain developments
to create an extremely secure consensus algorithm. All nodes vote
in multiple rounds, and only produce blocks when they are guaranteed
that the block is the "correct" globally consensus. Even in the case
of omnipotent network manipulation, this algorithm will never produce
to blocks at the same height (a fork) if less than one third of the
nodes are actively collaborating to break the system. This is possibly the
strongest guarantee of any production blockchain consensus algorithm.

The benefit of this approach is that any block that has over two thirds
of the signature is `provably correct by light clients <https://blog.cosmos.network/light-clients-in-tendermint-consensus-1237cfbda104>`__
The state is never rolled back and clients can take actions based on that
state. This opens the possibility of blockchain payments to be settled
in the order of a second or two, similar latency with using a credit
card in a store. It also allows reasonably responsive applications to
be built on a blockchain.
