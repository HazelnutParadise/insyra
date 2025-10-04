# GPArotation::vgQ.bigeomin  (namespace: GPArotation)
# dumped at 2025-10-04 13:16:30.691491
vgQ.bigeomin <- 
function (L, delta = 0.01) 
{
    Lg <- L[, -1, drop = FALSE]
    out <- vgQ.geomin(Lg, delta = delta)
    list(Gq = cbind(0, out$Gq), f = out$f, Method = "Bi-Geomin")
}

