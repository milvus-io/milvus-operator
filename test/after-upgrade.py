# Copyright (C) 2019-2020 Zilliz. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance
# with the License. You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software distributed under the License
# is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
# or implied. See the License for the specific language governing permissions and limitations under the License.


import random
import numpy as np
import time
import argparse
from pymilvus import (
    connections, list_collections,
    FieldSchema, CollectionSchema, DataType,
    Collection
)
TIMEOUT = 120


def hello_milvus(host="127.0.0.1"):
    import time
    # create connection
    connections.connect(host=host, port="19530")

    print(f"\nList collections...")
    print(list_collections())

    # create collection
    dim = 128
    default_fields = [
        FieldSchema(name="int64", dtype=DataType.INT64, is_primary=True),
        FieldSchema(name="float", dtype=DataType.FLOAT),
        FieldSchema(name="varchar", dtype=DataType.VARCHAR, max_length=65535),
        FieldSchema(name="float_vector", dtype=DataType.FLOAT_VECTOR, dim=dim)
    ]
    default_schema = CollectionSchema(fields=default_fields, description="test collection")

    print(f"\nCreate collection...")
    collection = Collection(name="hello_milvus", schema=default_schema)

    print(f"\nList collections...")
    print(list_collections())

    nb = 3000
    vectors = [[random.random() for _ in range(dim)] for _ in range(nb)]

    t0 = time.time()
    print(f"\nGet collection entities...")
    print(collection.num_entities)
    t1 = time.time()
    print(f"\nGet collection entities cost {t1 - t0:.4f} seconds")

    # create index and load table
    print(f"\nload collection...")
    t0 = time.time()
    collection.load()
    t1 = time.time()
    print(f"\nload collection cost {t1 - t0:.4f} seconds")

    # load and search
    topK = 5
    search_params = {"metric_type": "L2", "params": {"nprobe": 10}}
    t0 = time.time()
    print(f"\nSearch...")
    # define output_fields of search result
    res = collection.search(
        vectors[-2:], "float_vector", search_params, topK,
        "int64 > 100", output_fields=["int64", "float"], timeout=TIMEOUT
    )
    t1 = time.time()
    print(f"search cost  {t1 - t0:.4f} seconds")
    # show result
    for hits in res:
        for hit in hits:
            # Get value of the random value field for search result
            print(hit, hit.entity.get("float"))

    # query
    expr = "int64 in [2,4,6,8]"
    output_fields = ["int64", "float"]
    res = collection.query(expr, output_fields, timeout=TIMEOUT)
    sorted_res = sorted(res, key=lambda k: k['int64'])
    for r in sorted_res:
        print(r)
    # collection.release()

    COLOR = "\033[93m{}\033[0m"
    indexes = collection.indexes
    for index in indexes:
        print(COLOR.format(index.to_dict()))
    has = collection.has_index()
    print(COLOR.format(f"has index: {has}"))


parser = argparse.ArgumentParser(description='host ip')
parser.add_argument('--host', type=str, default='127.0.0.1', help='host ip')
args = parser.parse_args()
# add time stamp
print(f"\nStart time: {time.strftime('%Y-%m-%d %H:%M:%S', time.localtime(time.time()))}")
hello_milvus(args.host)

