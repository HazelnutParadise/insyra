# Generate test data matching Holzinger-Swineford dataset structure
# This is a well-known dataset for factor analysis

library(psych)

# Use built-in Harman data or create synthetic data
# The Holzinger-Swineford dataset has been used extensively
# We'll create a synthetic version with similar structure

set.seed(123)
n <- 20

# Two latent factors
f1 <- rnorm(n)
f2 <- rnorm(n)

# 8 variables: first 4 load on f1, last 4 on f2
x1 <- 0.7*f1 + 0.2*f2 + 0.5*rnorm(n)
x2 <- 0.6*f1 + 0.2*f2 + 0.6*rnorm(n)
x3 <- 0.5*f1 + 0.1*f2 + 0.7*rnorm(n)
x4 <- 0.4*f1 + 0.2*f2 + 0.7*rnorm(n)

x5 <- 0.2*f1 + 0.8*f2 + 0.4*rnorm(n)
x6 <- 0.2*f1 + 0.7*f2 + 0.5*rnorm(n)
x7 <- 0.1*f1 + 0.6*f2 + 0.6*rnorm(n)
x8 <- 0.2*f1 + 0.5*f2 + 0.7*rnorm(n)

data <- data.frame(x1, x2, x3, x4, x5, x6, x7, x8)

# Save the data
write.csv(data, "fa_test_dataset.csv", row.names=FALSE)

# Run factor analysis with PAF and Promax
fa_result <- fa(data, nfactors=2, fm="pa", rotate="promax", scores=FALSE, max.iter=200)

cat("\n=== PAF Results from R's psych::fa ===\n")
cat("\nLoadings:\n")
print(fa_result$loadings)

cat("\nCommunalities:\n")
print(fa_result$communality)

cat("\nStructure:\n")
print(fa_result$Structure)

cat("\nPhi:\n")
print(fa_result$Phi)
