# `account`


Action                  | Has Superuser | Domain Admin  | Account Owner | Anyone
--                      | --            | --            | --            | --
Registar a domain       | yes           | no            | no            | yes
Registar a domain       | no            | no            | no            | no
Change domain admin     | yes           | yes           | no            | no
Change domain admin     | no            | no            | no            | no
Renew a domain          | no            | yes           | yes           | yes
Renew a domain          | yes           | yes           | yes           | yes
Delete a domain         | yes           | yes           | no            | no
Delete a domain         | no            | no            | no            | no
Register an account     | yes           | yes           | no            | no
Register an account     | no            | no            | no            | yes
Renew an account        | no            | yes           | yes           | yes
Renew an account        | yes           | yes           | yes           | yes
Change account owner    | yes           | yes           | no            | no
Change account owner    | no            | no            | yes           | no
Change account targets  | yes           | no            | yes           | no
Change account targets  | no            | no            | yes           | no
Delete an account       | yes           | yes           | yes           | no
Delete an account       | no            | no            | yes           | no
