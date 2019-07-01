/*
Package distribution implements a revenue stream that is periodically distributing
collected coins between defined destinations.

Revenue instance defines destinations of an income. Coins are send to a revenue
account. Upon request or configuration change collected coins are distributed
between destinations.
Share of the income is declared by using destination weights.
Each destination is ensured to be paid before the configuration is changed. This
means it is not possible to for example remove a destination before distributing
collected funds.
Only an admin can alter a revenue configuration. It is a good idea to use a
multisig contract as an admin address value.

This functionality can be used to pay validators for their work. It is a
transparent and trustful way to split income.

*/
package distribution
