# GPArotation::vgQ.mccammon  (namespace: GPArotation)
# dumped at 2025-10-04 13:16:30.69886
vgQ.mccammon <- 
function (L) 
{
    k <- ncol(L)
    p <- nrow(L)
    S <- L^2
    M <- matrix(1, p, p)
    s2 <- colSums(S)
    P <- S/matrix(rep(s2, p), ncol = k, byrow = T)
    Q1 <- -sum(P * log(P))
    H <- -(log(P) + 1)
    R <- M %*% S
    G1 <- H/R - M %*% (S * H/R^2)
    s <- sum(S)
    p2 <- s2/s
    Q2 <- -sum(p2 * log(p2))
    h <- -(log(p2) + 1)
    alpha <- h %*% p2
    G2 <- rep(1, p) %*% t(h)/s - as.vector(alpha) * matrix(1, 
        p, k)
    Gq <- 2 * L * (G1/Q1 - G2/Q2)
    f <- log(Q1) - log(Q2)
    list(f = f, Gq = Gq, Method = "McCammon entropy")
}

