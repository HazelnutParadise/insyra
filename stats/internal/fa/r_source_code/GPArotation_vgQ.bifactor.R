# GPArotation::vgQ.bifactor  (namespace: GPArotation)
# dumped at 2025-10-04 13:16:30.690866
vgQ.bifactor <- 
function (L) 
{
    k <- ncol(L)
    Lt <- L[, 2:k]
    Lt2 <- Lt^2
    N <- matrix(1, nrow = k - 1, ncol = k - 1) - diag(k - 1)
    f <- sum(Lt2 * (Lt2 %*% N))
    Gt <- 4 * Lt * (Lt2 %*% N)
    G <- cbind(0, Gt)
    list(f = f, Gq = G, Method = "Bifactor Biquartimin")
}

