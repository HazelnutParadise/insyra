# GPArotation::vgQ.tandemII  (namespace: GPArotation)
# dumped at 2025-10-04 13:16:30.710817
vgQ.tandemII <- 
function (L) 
{
    LL <- (L %*% t(L))
    LL2 <- LL^2
    f <- sum(diag(crossprod(L^2, (1 - LL2) %*% L^2)))
    Gq1 <- 4 * L * ((1 - LL2) %*% L^2)
    Gq2 <- 4 * (LL * (L^2 %*% t(L^2))) %*% L
    Gq <- Gq1 - Gq2
    list(Gq = Gq, f = f, Method = "Tandem II")
}

