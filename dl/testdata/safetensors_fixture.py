import sys

import numpy as np
from safetensors.numpy import save_file


save_file(
    {
        "weights": np.array(
            [[1.25, -2.5, 3.75], [4.5, 0.0, -6.25]], dtype=np.float32
        ),
        "indices": np.array([[0, 3], [8, -1]], dtype=np.int64),
        "mask": np.array(
            [[True, False, True], [False, True, False]], dtype=np.bool_
        ),
    },
    sys.argv[1],
    metadata={"format": "pt", "note": "ignored by dl"},
)
