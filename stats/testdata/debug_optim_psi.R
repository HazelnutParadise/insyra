#!/usr/bin/env Rscript
# Reproduces near_collinear data and dumps R's L-BFGS-B optim trajectory
# for ML factor analysis so we can compare to Go's psi.

suppressMessages(library(psych))

# Reproduce generatedNearCollinearRows() from factor_analysis_test.go
n <- 30
rows <- matrix(NA_real_, nrow=n, ncol=5)
for (i in 0:(n-1)) {
  x <- as.double(i)
  base <- 1.20*sin(0.31*x) + 0.42*cos(0.19*x)
  other <- 0.85*sin(0.57*x+0.3) - 0.30*cos(0.41*x)
  rows[i+1, 1] <- 0.5 + base + 0.0021*sin(13.7*x)
  rows[i+1, 2] <- 0.5 + base + 0.0023*cos(11.3*x)
  rows[i+1, 3] <- -0.7 + 0.78*base + 0.62*sin(1.71*x)
  rows[i+1, 4] <- 1.4 + 0.71*other + 0.58*cos(2.13*x)
  rows[i+1, 5] <- -0.9 + 0.36*base + 0.41*other + 0.55*sin(2.91*x)
}
m <- rows

R_corr <- cor(m)
cat("R_correlation_matrix:\n")
print(R_corr, digits=18)

# Mirror psych::fa ML start
SMC <- psych::smc(R_corr)
cat("\nR_SMC:\n")
print(SMC, digits=18)

start <- 1 - SMC
start[start < 0.005] <- 0.005
start[start > 1] <- 1
cat("\nR_psi_init:\n")
print(start, digits=18)

# Define FAfn / FAgr identical to psych::fa internals
FAfn <- function(Psi, S, q) {
  sc <- diag(1/sqrt(Psi))
  Sstar <- sc %*% S %*% sc
  E <- eigen(Sstar, symmetric = TRUE, only.values = TRUE)
  e <- E$values[-(1L:q)]
  e <- sum(log(e) - e) - q + nrow(S)
  -e
}

FAgr <- function(Psi, S, q) {
  sc <- diag(1/sqrt(Psi))
  Sstar <- sc %*% S %*% sc
  E <- eigen(Sstar, symmetric = TRUE)
  L <- E$vectors[, 1L:q, drop=FALSE]
  load <- L %*% diag(sqrt(pmax(E$values[1L:q] - 1, 0)), q)
  load <- diag(sqrt(Psi)) %*% load
  g <- load %*% t(load) + diag(Psi) - S
  diag(g)/Psi^2
}

# Compare initial gradient
cat("\nR_initial_gradient at psi_init:\n")
g_init <- FAgr(start, R_corr, 2)
print(g_init, digits=18)

# Compare initial f
cat("R_initial_f:\n")
cat(format(FAfn(start, R_corr, 2), digits=18), "\n")

# Compute Sstar and eigenvalues at psi_init
sc_init <- diag(1/sqrt(start))
Sstar_init <- sc_init %*% R_corr %*% sc_init
cat("\nR_Sstar at psi_init:\n")
print(Sstar_init, digits=18)
E_init <- eigen(Sstar_init, symmetric = TRUE)
cat("\nR_eigenvalues at psi_init (descending):\n")
print(E_init$values, digits=18)

# Inject a callback to log every iteration
trace_psi <- list()
trace_f <- list()
fn_count <- 0

FAfn_traced <- function(Psi, S, q) {
  v <- FAfn(Psi, S, q)
  fn_count <<- fn_count + 1
  trace_psi[[length(trace_psi) + 1]] <<- Psi
  trace_f[[length(trace_f) + 1]] <<- v
  v
}

res <- optim(start, FAfn_traced, FAgr, method="L-BFGS-B",
             lower=0.005, upper=1,
             control=list(fnscale=1, parscale=rep(0.01, length(start))),
             S=R_corr, q=2)

cat("\n--- R optim result ---\n")
cat("convergence:", res$convergence, "\n")
cat("counts:", res$counts, "\n")
cat("R_optim_psi (raw output):\n")
print(res$par, digits=18)
cat("R_optim_f:\n")
cat(format(res$value, digits=18), "\n")

# Compare against Go's reported optim psi
go_psi <- c(0.005, 0.005, 0.2206429442705259, 0.5959349119627464, 0.2997013842577122)
cat("\nGO_optim_psi (from trace test):\n")
print(go_psi, digits=18)
cat("delta (R - Go):\n")
print(res$par - go_psi, digits=12)
