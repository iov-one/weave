# Sometimes its a README fix, or something like that - which isn't relevant for
# including in a CHANGELOG for example
has_code_changes = !git.modified_files.grep(/\.go$/).empty?

# Let people say that this isn't worth a CHANGELOG entry in the PR if they choose
declared_trivial = github.pr_body.include?("#trivial") || github.pr_body.include?("!nochangelog") || !has_code_changes

if !git.modified_files.include?("CHANGELOG.md") && !declared_trivial
  fail("Please include a CHANGELOG entry. \nYou can find it at [CHANGELOG.md](https://github.com/iov-one/weave/blob/master/CHANGELOG.md).", sticky: false)
end
