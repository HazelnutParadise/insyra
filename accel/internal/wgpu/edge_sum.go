package wgpu

import (
	"fmt"
	"math"
)

// edgeSumRoundedKernel sums, for each output i = b*nodes+t, the exact products
// weights[e]*vectors[b*nodes+ends[e]] over the edges e listed for t in
// edges[offsets[t]:offsets[t+1]], and rounds once. Grouped by target with
// ends = sources it is nn's EdgeSum; grouped by source with ends = targets it
// is the gradient of the values. params = nodes, total, 0, stride.
var edgeSumRoundedKernel = &kernel{name: "rounded-edge-sum", nRead: 5, nWrite: 1, body: `
struct Params { nodes: u32, total: u32, unused: u32, stride: u32, }
@group(0) @binding(0) var<storage, read> offsets: array<u32>;
@group(0) @binding(1) var<storage, read> edges: array<u32>;
@group(0) @binding(2) var<storage, read> ends: array<u32>;
@group(0) @binding(3) var<storage, read> weights: array<u32>;
@group(0) @binding(4) var<storage, read> vectors: array<u32>;
@group(0) @binding(5) var<storage, read_write> c: array<u32>;
@group(0) @binding(6) var<uniform> p: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let i = gid.x + gid.y * p.stride;
    if (i >= p.total) {
        return;
    }
    let row = (i / p.nodes) * p.nodes;
    let node = i % p.nodes;
    var r: ExactRegister;
    exact_clear(&r);
    for (var k: u32 = offsets[node]; k < offsets[node + 1u]; k = k + 1u) {
        let e = edges[k];
        exact_add_product(&r, weights[e], vectors[row + ends[e]]);
    }
    c[i] = exact_round(r);
}
`}

// edgeSumWeightsRoundedKernel gives the gradient of each edge's weight: the
// exact sum over the batch of upstream[b*nodes+targets[e]] *
// values[b*nodes+sources[e]], rounded once. params = edges, nodes, batch, stride.
var edgeSumWeightsRoundedKernel = &kernel{name: "rounded-edge-sum-weights", nRead: 4, nWrite: 1, body: `
struct Params { edges: u32, nodes: u32, batch: u32, stride: u32, }
@group(0) @binding(0) var<storage, read> sources: array<u32>;
@group(0) @binding(1) var<storage, read> targets: array<u32>;
@group(0) @binding(2) var<storage, read> values: array<u32>;
@group(0) @binding(3) var<storage, read> upstream: array<u32>;
@group(0) @binding(4) var<storage, read_write> c: array<u32>;
@group(0) @binding(5) var<uniform> p: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let e = gid.x + gid.y * p.stride;
    if (e >= p.edges) {
        return;
    }
    let s = sources[e];
    let t = targets[e];
    var r: ExactRegister;
    exact_clear(&r);
    for (var b: u32 = 0u; b < p.batch; b = b + 1u) {
        exact_add_product(&r, upstream[b * p.nodes + t], values[b * p.nodes + s]);
    }
    c[e] = exact_round(r);
}
`}

// EdgeSumRounded sums, for every row b of vectors (shaped [batch,nodes]) and
// every node, the products of weights and vectors along the edges grouped
// for that node by offsets and edges, ending at ends. The caller groups the
// edges; every index in them must lie in range.
func EdgeSumRounded(offsets, edges, ends, weights, vectors *Buffer, nodes, batch int) (*Buffer, error) {
	if nodes < 1 || batch < 0 || offsets.Len() != nodes+1 {
		return nil, fmt.Errorf("wgpu: rounded edge sum needs nodes >= 1, batch >= 0 and nodes+1 offsets, got nodes %d, batch %d, %d offsets", nodes, batch, offsets.Len())
	}
	if edges.Len() != ends.Len() || weights.Len() != ends.Len() {
		return nil, fmt.Errorf("wgpu: rounded edge sum has %d grouped edges, %d endpoints and %d weights", edges.Len(), ends.Len(), weights.Len())
	}
	if batch > math.MaxUint32/nodes || vectors.Len() != batch*nodes {
		return nil, fmt.Errorf("wgpu: rounded edge sum vectors hold %d values, want %d by %d", vectors.Len(), batch, nodes)
	}
	total := batch * nodes
	out, err := NewBuffer(total)
	if err != nil {
		return nil, err
	}
	if total == 0 {
		return out, nil
	}
	gx, gy, stride := elementwiseGroups(total)
	params := [16]uint32{uint32(nodes), uint32(total), 0, stride}
	if err := edgeSumRoundedKernel.run([]*Buffer{offsets, edges, ends, weights, vectors}, []*Buffer{out}, params, gx, gy); err != nil {
		out.Release()
		return nil, err
	}
	return out, nil
}

// EdgeSumWeightsRounded gives the gradient of every edge weight from values
// and upstream, both shaped [batch,nodes].
func EdgeSumWeightsRounded(sources, targets, values, upstream *Buffer, nodes, batch int) (*Buffer, error) {
	if nodes < 1 || batch < 0 || sources.Len() != targets.Len() {
		return nil, fmt.Errorf("wgpu: rounded edge-weight gradient needs nodes >= 1, batch >= 0 and one target per source, got nodes %d, batch %d, %d sources, %d targets", nodes, batch, sources.Len(), targets.Len())
	}
	if batch > math.MaxUint32/nodes || values.Len() != batch*nodes || upstream.Len() != batch*nodes {
		return nil, fmt.Errorf("wgpu: rounded edge-weight gradient has %d values and %d upstream values, want %d by %d", values.Len(), upstream.Len(), batch, nodes)
	}
	edges := sources.Len()
	out, err := NewBuffer(edges)
	if err != nil {
		return nil, err
	}
	if edges == 0 {
		return out, nil
	}
	gx, gy, stride := elementwiseGroups(edges)
	params := [16]uint32{uint32(edges), uint32(nodes), uint32(batch), stride}
	if err := edgeSumWeightsRoundedKernel.run([]*Buffer{sources, targets, values, upstream}, []*Buffer{out}, params, gx, gy); err != nil {
		out.Release()
		return nil, err
	}
	return out, nil
}
