------------------
Reacting to Events
------------------

**TODO**

We want to know when someone sends us tokens, but don't
want to constantly poll the blockchain (which will DDoS it
if enough clients do this). The solution is to subscribe
to events. Weave exposes a subscription event to listen
for any tx that modifies an account (reduce or increase
the balance).

* Subscribe to any tx that affect any accounts in local keybase
* Refresh balance page and post notification when balance changes
* Send a test tx from one client and observe change on another
* Update the transaction history
