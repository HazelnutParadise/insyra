# Reference values for wright_variance_ratio_test.go, from R vrtest 1.2
# (vrtest::Wright). Run with Rscript; the printed series and statistics are
# copied into the test.
library(vrtest)
set.seed(11)
y <- round(rnorm(80, 0, 0.01), 3)   # ties and zeros
cat("zeros:", sum(y == 0), " distinct:", length(unique(y)), "\n")
cat("y <- c(", paste(format(y, digits=17), collapse=", "), ")\n")
w <- Wright(y, c(2, 5, 10))$Stats
print(format(w, digits=17))
set.seed(5)
m <- cumsum(rnorm(60)) * 0.001 + rnorm(60) * 0.0005   # trending increments
w2 <- Wright(m, c(2, 4))$Stats
cat("m <- c(", paste(format(m, digits=17), collapse=", "), ")\n")
print(format(w2, digits=17))
