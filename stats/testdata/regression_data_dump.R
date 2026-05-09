## Dumps rnorm-generated data for regression diverse cases.
set.seed(2001); xL <- round(rnorm(50, 50, 10), 4)
yL <- 2.5 + 1.3 * xL + round(rnorm(50, 0, 5), 4)
cat("ln_largeN_x:", paste(xL, collapse=","), "\n", sep="")
cat("ln_largeN_y:", paste(yL, collapse=","), "\n", sep="")

set.seed(2002)
n <- 30
mx1 <- round(rnorm(n, 10, 2), 4)
mx2 <- round(rnorm(n, 5, 1.5), 4)
mx3 <- round(rnorm(n, 0, 1), 4)
my <- 2 + 1.5 * mx1 + 0.7 * mx2 - 0.3 * mx3 + round(rnorm(n, 0, 0.5), 4)
cat("ml_3pred_x1:", paste(mx1, collapse=","), "\n", sep="")
cat("ml_3pred_x2:", paste(mx2, collapse=","), "\n", sep="")
cat("ml_3pred_x3:", paste(mx3, collapse=","), "\n", sep="")
cat("ml_3pred_y:",  paste(my,  collapse=","), "\n", sep="")

set.seed(2003)
xp <- 1:15
yp <- round(0.5 * xp^4 - 2 * xp^3 + 3 * xp^2 - xp + 1 + rnorm(15, 0, 5), 3)
cat("pl_deg4_x:", paste(xp, collapse=","), "\n", sep="")
cat("pl_deg4_y:", paste(yp, collapse=","), "\n", sep="")

set.seed(2004); xN <- round(seq(1, 20, length.out = 25), 4)
yN <- round(3 + 2 * log(xN) + rnorm(25, 0, 0.5), 4)
cat("lo_noisy_x:", paste(xN, collapse=","), "\n", sep="")
cat("lo_noisy_y:", paste(yN, collapse=","), "\n", sep="")
