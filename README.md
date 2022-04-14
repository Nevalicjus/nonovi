# Nonovi

Nonovi is a terminal nonogram game


## nnv
Nonovi uses format **nnv** to store your nonograms.
Your nnv file should start with `b-` which signifies start of the board.
```
b-
```

Then, for every row in your board you should have a row's length of characters - `x` if it's filled, `_` if it's empty.
```
b-
xxx
x_x
xxx
```

If you're done with the board, your next line should be `eb-` to mark **e**nd of the **b**oard.
```
b-
xxx
x_x
xxx
eb-
```

Then, you should have `h-` to mark start of hints and two hint rows. First includes hints for columns, second - hints for rows.
Every set of hints should be separated by `:`, whilst each hint if multiple per row/column should be separated by `-`.
```
b-
xxx
x_x
xxx
eb-
h-
3:1-1:3
3:1-1:3
```

The next line after hints there should be `eh-` to mark **e**nd of **h**ints.
```
b-
xxx
x_x
xxx
eb-
h-
3:1-1:3
3:1-1:3
eh-
```

Et voilà! Your nnv file is ready to be played with nonovi.
