# Weave Security Audit Reports

This file contains a brief summary of latest security audits applied to `Weave` by trusted third parties.

## Scrutinized modules and aspects

- General Conditions / Permission system design
- Pieces related to money (and int overflow checks)
- Input parsing and validations (buffer overflow, etc)
- [x/sigs](./x/sigs)
- [x/multisig](./x/multisig)
- [x/gov](./x/gov)
- [x/cash](./x/cash)
- [x/distribution](./x/distribution)
- [x/escrow](./x/escrow)

## Report Summary

In every audit, IOV is praised by high code quality standards.
Observations of the audit firm on Weave:

> Our general observations of the code concluded that the overall quality is very high and well structured. The design decision to develop the application layer in Go has had a great impact on security since the inherited security implications of the Go language makes it hard to write unsecure code. The design decision to use the well proven and heavily reviewed [Tendermint Consensus Engine](https://tendermint.com/) limits a lot of potential vulnerabilities. Since both the Tendermint core and Weave are written in the Go programming language, the integration code is minimal, and the integration interfaces are clean. This design decision in conjunction with the strict development environment that requires linters and security checks before building and committing any code has led to an overall code quality that is very good both from both a security as well as from a “code smell” hygiene and readability standpoint. Furthermore, the use of [gogo](https://github.com/gogo/protobuf), a protobuf implementation for network serialization, meant that many of the normal security problems (e.g buffer overruns and input validation problems) have been effectively eliminated.

## Possible security vulnurabilities

### Regular expression denial of service in router component

_From the report:_
> Performance risks in HTTP router implementation - The HTTP router implementation uses a regular expression to parse the request string at [weave/router](https://github.com/iov-one/weave/app/router.go#L12). Regular Expressions potentially expose the application to computational denial of service via specially crafted inputs. This is also known as [ReDoS](https://www.owasp.org/index.php/Regular_expression_Denial_of_Service_-_ReDoS)

**Probability of Attack**: Low

**Ease of Exploitation**: Easy

This issue is dismissed by one of our engineers with this comment:
> Regular expression is used only during the application initialization, when [the `Router.Handler` is called](https://github.com/iov-one/weave/blob/master/app/router.go#L37-L42). When handling a message, the lookup is done with O(1) complexity, because [we are using an index](https://github.com/iov-one/weave/blob/master/app/router.go#L53-L54).

So no action is required about the issue.

### Critical function PrivKeyEd25519FromSeed is missing testcase

_From the report:_
> The function PrivKeyEd25519FromSeed that is critical for the generation of wallets is not covered by any unit test and could therefore be prone to future errors being implemented without being detected.

**Probability of Attack**: Low

**Ease of Exploitation**: Difficult

This issue has been resolved by this [PR](https://github.com/iov-one/weave/pull/1038).

**Audit firm detected only 2 medium level possible security flaws and these issues already have been resolved.**
