For v0 this architecture of Lexer is suficient.
We can extend it to supportmore features.

Current Lexer will start to hurt us when we begin work on stdc.
With #include, all file have to be parsed again,
even if behind #ifndef guard.

## Knowledge gain after initial implementation

Only few tokens had to be parsed before preprocesing.

Obvious ones like comments, string literals and preprocesor directives itself.
Not obvious one is character constant.

```c
int i_got_paid_by_a_line = '\
a\
b\
c\
d\
' // and thats a valid, standard conformant character

char thats_not_a_macro = '\
#'define PASS FAIL
```

Obvious things that don't have to be parsed includes
integer and floats constants. Not obvious ones:

```c
char/* comment */foo // thats still two idents.

</**/< // undefined behaviour in specs, so neither '<' '<' nor '<<'
// and other punctuations

#define GLUE(a,b) a ## b
GLUE(foo,13) // thats one token
bar/**/13 // and thats two
```

I especially looking forward to not parsing punctuations and floats from io.Reader.
Covering edge cases with ReadByte/UnreadByte was hard for `%:%:` or `0x1.1P+1f`.
