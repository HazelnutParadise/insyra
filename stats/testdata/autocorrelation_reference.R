## Reference values for the portmanteau statistics of stats/autocorrelation.go.
## Generated with R 4.6.1.
## Run from stats/testdata: Rscript autocorrelation_reference.R > autocorrelation_reference.txt
## Q and PQ come from base R's Box.test(type = "Ljung-Box", fitdf = 0).
## QRobust is computed from its definition, the robust Box-Pierce statistic of
## Lobato, Nankervis and Savin (2001) as written by Escanciano and Lobato
## (2009): QRobust = n * sum_{j=1..m} rho_j^2 / delta_j, with
## delta_j = n * sum_{t=j+1..n} d_t^2 d_{t-j}^2 / (sum d_t^2)^2 and d = x - mean(x).

options(digits = 17)
fmt <- function(x) formatC(x, digits = 17, format = "g")
emit <- function(key, val) cat(key, "=", fmt(val), "\n", sep = "")

## The series of TestAutocorrelationMatchesR.
x <- sin(1:30) * cos((1:30)^2 / 7) + (1:30) %% 3
## A longer series with clustered volatility.
set.seed(499)
e <- rnorm(200)
s <- exp(0.8 * sin((1:200) / 15))
y <- e * s + 0.3 * c(0, head(e * s, -1))

emit_set <- function(prefix, x, maxLag) {
  n <- length(x)
  d <- x - mean(x)
  ss <- sum(d^2)
  qr <- 0
  for (m in 1:maxLag) {
    b <- Box.test(x, lag = m, type = "Ljung-Box", fitdf = 0)
    emit(sprintf("%s.Q[%d]", prefix, m), unname(b$statistic))
    emit(sprintf("%s.PQ[%d]", prefix, m), b$p.value)
    rho <- sum(d[(m + 1):n] * d[1:(n - m)]) / ss
    delta <- n * sum(d[(m + 1):n]^2 * d[1:(n - m)]^2) / ss^2
    qr <- qr + n * rho^2 / delta
    emit(sprintf("%s.QRobust[%d]", prefix, m), qr)
    emit(sprintf("%s.PQRobust[%d]", prefix, m), pchisq(qr, df = m, lower.tail = FALSE))
  }
}
emit_set("short", x, 4)
emit_set("clustered", y, 10)
## The clustered series itself, so the Go test reads exactly what R used.
for (i in seq_along(y)) emit(sprintf("clustered.x[%d]", i), y[i])
