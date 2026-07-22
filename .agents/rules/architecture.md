---
trigger: always_on
---

Tusk Architecture Philosophy

Tusk should feel like Go, not like Laravel, Spring Boot, or ASP.NET.

Every package should have one clear responsibility.

Every module should own its business domain.

Avoid deep abstractions.

Avoid framework magic.

Avoid hidden behavior.

Avoid reflection unless absolutely necessary.

Avoid code generation unless it clearly improves developer experience.

Every dependency should be visible in constructors.

Use dependency injection through constructors.

Keep APIs small.

Keep exported APIs intentional.

Write code that is obvious to read.

If a design becomes difficult to explain, it is probably too complicated.

Prefer standard library solutions over third-party packages unless the package provides significant value.

The framework should teach good Go by example.

Whenever you improve my code:

Do not only rewrite it.

Explain:

- why the original design was less idiomatic
- why the new design is preferred
- what Go principle it follows
- whether the change improves readability, simplicity, testing, or maintainability

Assume I am learning advanced Go architecture.

Prioritize helping me become a better Go engineer over simply producing code.

Before introducing any abstraction, ask:

- Does this abstraction reduce complexity?
- Or does it merely hide complexity?

Avoid abstractions that exist only for future possibilities.

Follow the Go principle:

"A little copying is better than a little dependency."

Prefer duplication over premature abstraction when it leads to simpler, more readable code.