# GPArotation::vgQ.lp.wls  (namespace: GPArotation)
# dumped at 2025-10-04 13:16:30.696676
vgQ.lp.wls <- 
function (L, W) 
{
    list(Gq = 2 * W * L/nrow(L), f = sum(W * L * L)/nrow(L), 
        Method = "Weighted least squares for Lp rotation")
}

