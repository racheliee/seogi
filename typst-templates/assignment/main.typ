#import "./lib.typ": assignment

#show: assignment.with(
  title: "Assignment 1",
  course: "SWE3003: Introduction to Real-time system",
  authors: (
    (
      name: "Hyungjun Shon",
      email: "example@gmail.com",
      student-no: "20xxxxxxxx",
    ),
  ),

  
)

== 일어날 수도 있던 일

- factorial 팩토리얼을 찾아라
- Find the factorial $n!$, of a positive integer $n$.\
- The factorial $n!$ is defined as: $ n! eq.def product_"i=1"^n i = 1 times 2 times
... times n $

== Solution

This is a fairly simple problem which can be trivially solved using loops. 한글 진짜 개못 생겼따.

```py
fact = 1
for i in range(1, n+1):
	fact *= i

print(fact)
```

We can approach the same thing using recursion also, by exploiting the fact
that $n! = n times (n-1)!$.

```py
def fact(n):
	if n > 1:
		return n*fact(n-1)
	return 1

print(fact(n))
```

Now coming to the main part, we can get our one-liner using the recursive
approach.

```py
def fact(n): return n*fact(n-1) if n > 1 else 1

print(fact(n))
```

#pagebreak()

그렇지 않아 2023년에는 question
#lorem(500)
