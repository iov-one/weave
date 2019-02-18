/*
Package feedist implements a revenue stream that is periodically distributing
collected coins between defined recipients.

Revenue instance defines recipients of an income. Coins are send to a revenue
account. Upon request or configuration change collected coins are distributed
between recipients.
Share of the income is declared by using recipient weights.
Each recipient is ensured to be paid before the configuration is changed. This
means it is not possible to for example remove a recipient before distributing
collected funds.
Only an admin can alter a revenue configuration. It is a good idea to use a
multisig contract as an admin address value.

This functionality can be used to pay validators for their work. It is a
transparent and trustful way to split income.

*/
package feedist
