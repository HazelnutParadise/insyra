# GPArotation::vgQ.oblimax  (namespace: GPArotation)
# dumped at 2025-10-04 13:16:30.700908
vgQ.oblimax <- 
function (L) 
{
    list(Gq = -(4 * L^3/(sum(L^4)) - 4 * L/(sum(L^2))), f = -(log(sum(L^4)) - 
        2 * log(sum(L^2))), Method = "Oblimax")
}

