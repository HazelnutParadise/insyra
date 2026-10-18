package stats

import (
	"math"
	"math/rand"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// varianceRatioSeries is 120 draws from N(0.0005, 0.01) (Python random, seed 3).
// The expected values below come from an independent loop-by-loop Python
// implementation of the Lo and MacKinlay (1988) definitions on the same numbers;
// R vrtest and arch were not installed to compare against.
var varianceRatioSeries = []any{
	0.0014470803828730422, 0.013000243810835505, -0.00881378367720707, 0.010423772805192404,
	-0.0020915453769343407, -0.0021151098398117397, 0.019497252784647572, 0.002075370716337192,
	7.075746207755684e-05, 0.0077949848643560915, 0.011768539623838552, 0.00019156724965069792,
	0.0063799374518036436, -0.009237243574656976, -0.0031679046998052923, -0.003881250344027819,
	-0.012822831645911484, -0.014585141271610723, -0.015769127299570283, -0.0018865285544380351,
	-0.0012242544785211128, -0.002703377073792124, 0.001191276447150572, -0.012855855015706947,
	-0.00029465031295354007, 0.002880985047634084, 0.008010424068935217, -0.007962226452665477,
	-0.003498708069755593, -0.01965177655919042, -0.0045364956388200175, -0.02146691021636984,
	-0.01369386020892132, 0.011515114923556098, -0.021516289594427317, 0.008485617632771144,
	0.0037789665192709317, -0.0026233909952836196, 0.005093434377705517, 0.005774612940411891,
	0.010954213122973976, -0.0018037016360335843, -0.005422198409312717, -0.005546153278229037,
	-0.00936441602155549, 5.077002509508622e-05, -0.007357585461573914, 0.01118594497766815,
	-0.018194236238317396, -0.010437447017902739, -0.009031743346778422, -0.020428635210550026,
	0.019522786173852908, -0.023583391886659814, -0.0023328885154105645, -0.004752122457411884,
	0.01705919906709646, -0.019354164492383367, 0.011218333483799898, -0.006814140888110802,
	-0.0010522468069788687, -0.006209694890761954, 0.006904752171816783, -0.010876418755214447,
	-0.00028898499202372017, 0.004034152669265362, 0.018902581640776725, -0.02355278518425364,
	0.015751235981050084, 0.009980695785399805, -0.004335881496406466, 0.0035492958673817715,
	-0.004160035504890591, 0.01696728513528854, 0.0025959040463540864, -0.0016555567981893656,
	-0.0017806193175072494, -0.0014987619091113929, -0.0012661837274387509, -0.00828983111703151,
	0.021083391855573137, -0.018609929086382055, -0.03555752521430666, -0.0007284937935559679,
	-0.0009656480232688202, 0.0042243438382709095, -0.001545056586675564, -0.0009811690203528315,
	0.003823455653233349, 0.010184151831691388, -0.0039797691621581595, -0.003228187892169132,
	0.01990524049369273, 0.005800263719124694, -0.009354495609923668, 0.023739429559340948,
	0.008259073163812464, -0.0053899969175147475, -0.011244141432077522, 0.003483184472756254,
	-0.007822028616880224, -0.01007538866928979, -0.012460009729596114, -0.004561753763086214,
	0.01157548908617568, -0.003835163372417297, -0.013994183450733418, 0.007188365112608508,
	0.0011569170601254898, 0.008938851927405497, 0.01251734968499696, -0.0011874239306383812,
	-0.0009483857645620551, 4.513244764394543e-05, -0.01084595599805146, 0.007189755739833326,
	0.014222106592390697, 0.002238138291769739, -0.0018743987073415584, -0.0020989309680350724,
}

func TestVarianceRatioMatchesTheDefinition(t *testing.T) {
	got, err := VarianceRatio(insyra.NewDataList(varianceRatioSeries...), 2, 5, 10)
	if err != nil {
		t.Fatal(err)
	}
	want := []struct{ vr, z, zStar float64 }{
		{0.9126530007358231, -0.9568384365475365, -0.8183751731038653},
		{1.103336831410324, 0.5166841570516201, 0.4678588109234243},
		{1.236230911590802, 0.7664342838655082, 0.7454708152199598},
	}
	for i, w := range want {
		r := got[i]
		if math.Abs(r.VR-w.vr) > 1e-11 || math.Abs(r.ZHomoscedastic-w.z) > 1e-10 || math.Abs(r.ZRobust-w.zStar) > 1e-10 {
			t.Errorf("q=%d: got %+v, want %+v", r.Q, r, w)
		}
		if math.Abs(r.PHomoscedastic-2*(1-zCDF(math.Abs(w.z)))) > 1e-10 {
			t.Errorf("q=%d: p-value %v", r.Q, r.PHomoscedastic)
		}
	}
}

func TestVarianceRatioHasItsNominalSizeAndDetectsMomentum(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	const reps, n = 1500, 400
	rejectHomo, rejectRobust := 0, 0
	for r := 0; r < reps; r++ {
		x := make([]any, n)
		for i := range x {
			x[i] = rng.NormFloat64()
		}
		res, err := VarianceRatio(insyra.NewDataList(x...), 5)
		if err != nil {
			t.Fatal(err)
		}
		if res[0].PHomoscedastic < 0.05 {
			rejectHomo++
		}
		if res[0].PRobust < 0.05 {
			rejectRobust++
		}
	}
	for name, count := range map[string]int{"homoscedastic": rejectHomo, "robust": rejectRobust} {
		if rate := float64(count) / reps; rate < 0.03 || rate > 0.075 {
			t.Errorf("%s z rejects %.3f of random walks, want about 0.05", name, rate)
		}
	}
	// Positively autocorrelated returns give VR(q) above 1 and a rejection.
	x := make([]any, 2000)
	prev := 0.0
	for i := range x {
		prev = 0.4*prev + rng.NormFloat64()
		x[i] = prev
	}
	res, _ := VarianceRatio(insyra.NewDataList(x...), 5)
	if res[0].VR <= 1.2 || res[0].PRobust > 0.001 {
		t.Errorf("momentum: %+v", res[0])
	}
}

func TestChowDenning(t *testing.T) {
	list := insyra.NewDataList(varianceRatioSeries...)
	cd, err := ChowDenning(list, false, 2, 5, 10)
	if err != nil {
		t.Fatal(err)
	}
	// The largest |z| of the three is 0.9568; Sidak with 3 periods.
	wantP := 1 - math.Pow(2*zCDF(0.9568384365475365)-1, 3)
	if math.Abs(cd.MaxZ-0.9568384365475365) > 1e-10 || math.Abs(cd.P-wantP) > 1e-10 {
		t.Errorf("got %+v, want P %v", cd, wantP)
	}
	robust, _ := ChowDenning(list, true, 2, 5, 10)
	if math.Abs(robust.MaxZ-0.8183751731038653) > 1e-10 {
		t.Errorf("robust: %+v", robust)
	}
}

func TestVarianceRatioRefusesBadInput(t *testing.T) {
	good := insyra.NewDataList(0.01, -0.02, 0.03, 0.0, 0.01, -0.01)
	var nilList *insyra.DataList
	cases := map[string]func() error{
		"no q":      func() error { _, e := VarianceRatio(good); return e },
		"q 1":       func() error { _, e := VarianceRatio(good, 1); return e },
		"q too big": func() error { _, e := VarianceRatio(good, 6); return e },
		"nil":       func() error { _, e := VarianceRatio(nilList, 2); return e },
		"NaN":       func() error { _, e := VarianceRatio(insyra.NewDataList(0.1, math.NaN(), 0.2), 2); return e },
		"constant":  func() error { _, e := VarianceRatio(insyra.NewDataList(1.0, 1.0, 1.0, 1.0), 2); return e },
		"cd no q":   func() error { _, e := ChowDenning(good, false); return e },
	}
	for name, f := range cases {
		if f() == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}
