## How multisig work in action
A brief summary with code links to the client and server side:

1. Create a multisig condition: https://github.com/iov-one/weave/blob/master/cmd/bnsd/scenarios/main_test.go#L56

2. Configure the multisig in the genesis: https://github.com/iov-one/weave/blob/master/cmd/bnsd/scenarios/main_test.go#L169

3. Submit the multisig contract ID by the **client**: https://github.com/iov-one/weave/blob/master/cmd/bnsd/scenarios/main_test.go#L169

4. The contract is loaded on the **server side**: https://github.com/iov-one/weave/blob/master/x/multisig/decorator.go#L45

5. Signatures are validated and threshold compared: https://github.com/iov-one/weave/blob/master/x/multisig/decorator.go#L69

    5b. https://github.com/iov-one/weave/blob/master/x/auth.go#L86

6. Added to the request context: https://github.com/iov-one/weave/blob/master/x/multisig/context.go#L24

7. Verified by a handler: https://github.com/iov-one/weave/blob/master/x/validators/handler.go#L14

   7c. In the multisig authenticator: https://github.com/iov-one/weave/blob/master/x/multisig/context.go#L51
    
   7b. Setup at app level: https://github.com/iov-one/weave/blob/master/cmd/bnsd/app/app.go#L37
    
