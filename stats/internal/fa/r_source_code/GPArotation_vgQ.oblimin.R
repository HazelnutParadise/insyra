# GPArotation::vgQ.oblimin  (namespace: GPArotation)
# dumped at 2025-10-04 13:16:30.702317
vgQ.oblimin <- 
function (L, gam = 0) 
{
    X <- L^2 %*% (!diag(TRUE, ncol(L)))
    if (0 != gam) {
        p <- nrow(L)
        X <- (diag(1, p) - matrix(gam/p, p, p)) %*% X
    }
    list(Gq = L * X, f = sum(L^2 * X)/4, Method = if (gam == 
        0) "Oblimin Quartimin" else if (gam == 0.5) "Oblimin Biquartimin" else paste("Oblimin g=", 
        gam, sep = ""))
}

