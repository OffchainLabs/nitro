---
status: accepted
date: 2024-11-29
decision-makers: eljobe@ plasmapower@
---

# Avoid primitive constraint types

## Context and Problem Statement

When working on the go code for BoLD, we became slightly annoyed that several
places in the history package were checking the constraint that the `virtual`
argumet to a function was positive. One possible workaround would have been
to create a constrained wrapper type around `uint64` which would only allow
positive values. For example:

```go
// Pos64 is a type which represents a positive uint64.
//
// The "zero" value of Pos64 is 1.
type Pos64 struct {
	uint64
}

// NewPos64 returns a new Pos64 with the given value.
//
// errors if v is 0.
func NewPos64(v uint64) (Pos64, error) {
	if v == 0 {
		return Pos64{}, errors.New("v must be positive. got: 0")
	}
	return Pos64{v}, nil
}

// MustPos64 returns a new Pos64 with the given value.
//
// panics if v is 0.
func MustPos64(v uint64) Pos64 {
	if v == 0 {
		panic("v must be positive. got: 0")
	}
	return Pos64{v}
}

// Val returns the value of the Pos64.
func (p Pos64) Val() uint64 {
	// The zero value of Pos64 is 1.
	if p.uint64 == 0 {
		return 1
	}
	return p.uint64
}
```

The idea being that within a package, all of the functions which needed to deal
with a `virtual` argument, could take in a `Pos64` instead of a `uint64` and it
would be up to clients of the package to ensure that they only passed in
positive values.

## Considered Options

* New Package: `util/chk` for checking type constraint
* Status Quo: check the constraint in multiple places
* Minimize Checks: no check in package private functions

## Decision Outcome

Chosen option: "Status Quo", because the "New Package" option introduces a
regression in being able to use type type with operators, and "Minimize Checks"
is too prone to bugs introduced by refactoring.


## Pros and Cons of the Options

### New Pacakge: `util/chk` for checking type constraint

* Good, because it is expressive
* Good, because the constraint only needs to be checked during construction
* Bad, because `Pos64` doesn't compile with operators like `+ * - /`

### Status Quo: check the constraint in multiple places

* Good, because it is what the code is already doing
* Good, because when a funciton becomes public, the constraint holds
* Good, because when a function moves to another file or package, the constraint holds
* Bad, because it means the check may need to be repeated. DRY

### Minimize Checks: no check in package private functions

* Good, because it reduces the amount of times a constraint is checked
* Bad, because the assumption might be violated if a private function becomes
  public, or gains an additional caller.

## More Information

See the discussion on now-closed #2743
