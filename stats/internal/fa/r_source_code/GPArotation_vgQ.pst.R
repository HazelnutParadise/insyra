# GPArotation::vgQ.pst  (namespace: GPArotation)
# dumped at 2025-10-04 13:16:30.704175
vgQ.pst <- 
function (L, W = NULL, Target = NULL) 
{
    if (is.null(W)) 
        stop("argument W must be specified.")
    if (is.null(Target)) 
        stop("argument Target must be specified.")
    Btilde <- W * Target
    list(Gq = 2 * (W * L - Btilde), f = sum((W * L - Btilde)^2), 
        Method = "Partially specified target")
}

