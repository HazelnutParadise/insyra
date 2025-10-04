# GPArotation::vgQ.geomin  (namespace: GPArotation)
# dumped at 2025-10-04 13:16:30.694362
vgQ.geomin <- 
function (L, delta = 0.01) 
{
    k <- ncol(L)
    p <- nrow(L)
    L2 <- L^2 + delta
    pro <- exp(rowSums(log(L2))/k)
    list(Gq = (2/k) * (L/L2) * matrix(rep(pro, k), p), f = sum(pro), 
        Method = "Geomin")
}

