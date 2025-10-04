# GPArotation::vgQ.infomax  (namespace: GPArotation)
# dumped at 2025-10-04 13:16:30.695325
vgQ.infomax <- 
function (L) 
{
    k <- ncol(L)
    p <- nrow(L)
    S <- L^2
    s <- sum(S)
    s1 <- rowSums(S)
    s2 <- colSums(S)
    E <- S/s
    e1 <- s1/s
    e2 <- s2/s
    Q0 <- sum(-E * log(E))
    Q1 <- sum(-e1 * log(e1))
    Q2 <- sum(-e2 * log(e2))
    f <- log(k) + Q0 - Q1 - Q2
    H <- -(log(E) + 1)
    alpha <- sum(S * H)/s^2
    G0 <- H/s - alpha * matrix(1, p, k)
    h1 <- -(log(e1) + 1)
    alpha1 <- s1 %*% h1/s^2
    G1 <- matrix(rep(h1, k), p)/s - as.vector(alpha1) * matrix(1, 
        p, k)
    h2 <- -(log(e2) + 1)
    alpha2 <- h2 %*% s2/s^2
    G2 <- matrix(rep(h2, p), ncol = k, byrow = T)/s - as.vector(alpha2) * 
        matrix(1, p, k)
    Gq <- 2 * L * (G0 - G1 - G2)
    list(f = f, Gq = Gq, Method = "Infomax")
}

