# GPArotation::vgQ.entropy  (namespace: GPArotation)
# dumped at 2025-10-04 13:16:30.693415
vgQ.entropy <- 
function (L) 
{
    list(Gq = -(L * log(L^2 + (L^2 == 0)) + L), f = -sum(L^2 * 
        log(L^2 + (L^2 == 0)))/2, Method = "Minimum entropy")
}

