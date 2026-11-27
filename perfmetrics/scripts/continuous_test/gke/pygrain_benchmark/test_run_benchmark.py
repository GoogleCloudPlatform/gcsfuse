#!/usr/bin/env python3
# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Unit tests for the GKE MaxText + PyGrain continuous benchmark suite."""

import asyncio
import os
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

SCRIPT_DIR = os.path.dirname(os.path.realpath(__file__))
if SCRIPT_DIR not in sys.path:
    sys.path.insert(0, SCRIPT_DIR)
if os.path.dirname(SCRIPT_DIR) not in sys.path:
    sys.path.insert(0, os.path.dirname(SCRIPT_DIR))

import generate_data
import maxtext_benchmark_runner
import run_benchmark


class TestParseAllGbytesPerSec(unittest.TestCase):
    """Tests for throughput extraction from pod logs."""

    def test_parse_multi_iteration_logs(self):
        logs = """
=== Starting MaxText + PyGrain Benchmark on TPU v6e-4 ===
[PyGrain] Total records discovered: 32768
Iteration 1/8: gbytes_per_sec: 14.85 Bytes/s (pygrain_gib_s=13.83 GiB/s, nic_gb_s=14.12 GB/s)
Iteration 2/8: gbytes_per_sec: 15.20 Bytes/s (pygrain_gib_s=14.16 GiB/s, nic_gb_s=14.40 GB/s)
Iteration 3/8: gbytes_per_sec: 13.95 Bytes/s (pygrain_gib_s=12.99 GiB/s, nic_gb_s=13.50 GB/s)
Iteration 4/8: gbytes_per_sec: 16.10 Bytes/s (pygrain_gib_s=14.99 GiB/s, nic_gb_s=15.05 GB/s)
"""
        extracted = run_benchmark.parse_all_gbytes_per_sec(logs)
        self.assertEqual(extracted, [14.85, 15.20, 13.95, 16.10])

    def test_parse_single_iteration_log(self):
        logs = "gbytes_per_sec: 13.42 Bytes/s\n"
        self.assertEqual(run_benchmark.parse_all_gbytes_per_sec(logs), [13.42])

    def test_parse_empty_or_malformed_logs(self):
        self.assertEqual(run_benchmark.parse_all_gbytes_per_sec(""), [])
        self.assertEqual(
            run_benchmark.parse_all_gbytes_per_sec(
                "Some unrelated log line\ngbytes_per_sec: invalid Bytes/s"
            ),
            [],
        )

    def test_parse_json_metrics_fallback(self):
        logs = """
===JSON_METRICS_START===
{
  "iteration_throughputs_gbps": [14.12, 14.55, 13.98, 15.01, 14.22, 13.87, 14.44, 14.09]
}
===JSON_METRICS_END===
"""
        self.assertEqual(
            run_benchmark.parse_all_gbytes_per_sec(logs),
            [14.12, 14.55, 13.98, 15.01, 14.22, 13.87, 14.44, 14.09],
        )


class TestPodTemplateSubstitution(unittest.TestCase):
    """Tests for pod.yaml.template variable rendering."""

    def test_render_pod_manifest_substitutes_all_variables(self):
        rendered = run_benchmark.render_pod_manifest(
            project_id="test-project-123",
            bucket_name="test-pygrain-bucket",
            iterations=8,
            staging_version="prow-gob-internal-boskos-pygrain-benchmark",
            client_protocol="http1",
        )
        self.assertNotIn(
            "$",
            rendered,
            "Rendered manifest must not contain any unreplaced $ template variables",
        )
        self.assertIn("name: gcsfuse-test", rendered)
        self.assertIn(
            "gcr.io/test-project-123/cloudbuild-gcsfuse-csi/gcs-fuse-csi-driver-sidecar-mounter:prow-gob-internal-boskos-pygrain-benchmark",
            rendered,
        )
        self.assertIn(
            "gcr.io/test-project-123/kislayk_orbax_workload:latest", rendered
        )
        self.assertIn('bucketName: "test-pygrain-bucket"', rendered)
        self.assertIn("client-protocol=http1", rendered)
        self.assertIn("--iterations=8", rendered)
        self.assertIn('hostNetworkPodKSA: "true"', rendered)

    def test_render_pod_manifest_grpc_protocol(self):
        rendered = run_benchmark.render_pod_manifest(
            project_id="gcs-fuse-test-ml",
            bucket_name="gcsfuse_gke_pygrain_benchmark_euw4",
            iterations=20,
            staging_version="prow-gob-internal-boskos-pygrain-benchmark",
            client_protocol="grpc",
        )
        self.assertNotIn("$", rendered)
        self.assertIn("client-protocol=grpc", rendered)
        self.assertIn("--iterations=20", rendered)


class TestEvaluateThroughputs(unittest.TestCase):
    """Tests for the 5/8 threshold evaluation rule and gRPC fallback."""

    def test_five_eighths_boundary_http1(self):
        # 5 out of 8 >= 13.0 -> pass
        passed, count = run_benchmark.evaluate_throughputs(
            [14.0, 13.5, 13.0, 15.1, 13.2, 12.9, 11.0, 10.5],
            threshold_gbps=13.0,
            client_protocol="http1",
        )
        self.assertTrue(passed)
        self.assertEqual(count, 5)

        # 4 out of 8 >= 13.0 -> fail for http1
        passed, count = run_benchmark.evaluate_throughputs(
            [14.0, 13.5, 13.0, 15.1, 12.9, 12.8, 11.0, 10.5],
            threshold_gbps=13.0,
            client_protocol="http1",
        )
        self.assertFalse(passed)
        self.assertEqual(count, 4)

    def test_twenty_iterations_boundary_http1(self):
        # 5/8 of 20 = 12.5 -> requires 13 successful iterations
        passed_13, count_13 = run_benchmark.evaluate_throughputs(
            [13.5] * 13 + [12.0] * 7,
            threshold_gbps=13.0,
            client_protocol="http1",
        )
        self.assertTrue(passed_13)
        self.assertEqual(count_13, 13)

        passed_12, count_12 = run_benchmark.evaluate_throughputs(
            [13.5] * 12 + [12.0] * 8,
            threshold_gbps=13.0,
            client_protocol="http1",
        )
        self.assertFalse(passed_12)
        self.assertEqual(count_12, 12)

    def test_grpc_warning_fallback_and_empty_failure(self):
        # 4/8 below threshold for grpc -> warn and pass
        passed_grpc, count_grpc = run_benchmark.evaluate_throughputs(
            [14.0, 13.5, 13.0, 15.1, 12.0, 11.5, 11.0, 10.5],
            threshold_gbps=13.0,
            client_protocol="grpc",
        )
        self.assertTrue(passed_grpc)
        self.assertEqual(count_grpc, 4)

        # Empty throughputs must fail even for grpc
        passed_empty, count_empty = run_benchmark.evaluate_throughputs(
            [], threshold_gbps=13.0, client_protocol="grpc"
        )
        self.assertFalse(passed_empty)
        self.assertEqual(count_empty, 0)


class TestDatasetExistenceAndGeneration(unittest.TestCase):
    """Tests for dataset existence checking and synthetic shard generation."""

    def test_check_dataset_exists_and_ensure_provisioned(self):
        shard_lines = "\n".join(
            f"gs://test-bucket/train-{i:05d}-of-00128.arrayrecord"
            for i in range(128)
        )

        async def run_test():
            with mock.patch.object(
                run_benchmark.utils,
                "run_command_async",
                new=mock.AsyncMock(return_value=(shard_lines, "", 0)),
            ):
                exists, cnt = await run_benchmark.check_dataset_exists(
                    "test-bucket", expected_shards=128
                )
                self.assertTrue(exists)
                self.assertEqual(cnt, 128)

                gen_mock = mock.MagicMock()
                triggered = await run_benchmark.ensure_dataset_provisioned(
                    "test-bucket", expected_shards=128, generator_fn=gen_mock
                )
                self.assertFalse(triggered)
                gen_mock.assert_not_called()

            # Incomplete shards (only 10 present) -> triggers generation
            incomplete_lines = "\n".join(
                f"gs://test-bucket/train-{i:05d}-of-00128.arrayrecord"
                for i in range(10)
            )
            with mock.patch.object(
                run_benchmark.utils,
                "run_command_async",
                new=mock.AsyncMock(return_value=(incomplete_lines, "", 0)),
            ):
                exists, cnt = await run_benchmark.check_dataset_exists(
                    "test-bucket", expected_shards=128
                )
                self.assertFalse(exists)
                self.assertEqual(cnt, 10)

                gen_mock = mock.MagicMock()
                triggered = await run_benchmark.ensure_dataset_provisioned(
                    "test-bucket", expected_shards=128, generator_fn=gen_mock
                )
                self.assertTrue(triggered)
                gen_mock.assert_called_once_with("test-bucket", 128)

        asyncio.run(run_test())

    def test_generate_data_record_and_shard_reuse(self):
        rec = generate_data.create_synthetic_vlm_record(
            seq_length=128, image_bytes_size=4096
        )
        # Header (16B) + tokens (512B) + positions (512B) + masks (128B) + image (4096B) = 5264B
        self.assertEqual(len(rec), 16 + 512 + 512 + 128 + 4096)

        unpacker = maxtext_benchmark_runner.UnpackMultimodalRecord(
            tiles_per_sample=1, compact_ipc=False
        )
        unpacked = unpacker.map(rec)
        self.assertEqual(len(unpacked["tokens"]), 128)
        self.assertEqual(len(unpacked["positions"]), 128)
        self.assertEqual(len(unpacked["image_masks"]), 128)
        self.assertEqual(len(unpacked["images"]), 4096)
        self.assertEqual(unpacked["payload_bytes"], len(rec))

        with tempfile.TemporaryDirectory() as tmpdir:
            self.assertEqual(
                generate_data.existing_shard_count(tmpdir, num_shards=4), 0
            )
            for i in range(4):
                shard_file = os.path.join(
                    tmpdir, f"train-{i:05d}-of-00004.arrayrecord"
                )
                with open(shard_file, "wb") as f:
                    f.write(b"synthetic-shard-bytes")
            self.assertEqual(
                generate_data.existing_shard_count(tmpdir, num_shards=4), 4
            )


class TestResolveStagingVersionAndCliArgs(unittest.TestCase):
    """Tests for CSI staging version fallback and all 16 CLI flags."""

    def test_resolve_staging_version_fallback(self):
        async def run_test():
            # When skip_csi_driver_build=False, returns STAGING_VERSION directly
            ver = await run_benchmark.resolve_staging_version(
                "gcs-fuse-test", skip_csi_driver_build=False
            )
            self.assertEqual(ver, run_benchmark.STAGING_VERSION)

            # When skip_csi_driver_build=True and image is missing (returncode=1), falls back
            with mock.patch.object(
                run_benchmark.utils,
                "run_command_async",
                new=mock.AsyncMock(return_value=("", "Not found", 1)),
            ):
                ver_fallback = await run_benchmark.resolve_staging_version(
                    "gcs-fuse-test", skip_csi_driver_build=True
                )
                self.assertEqual(
                    ver_fallback, run_benchmark.FALLBACK_STAGING_VERSION
                )

        asyncio.run(run_test())

    def test_cli_flags_and_help(self):
        parser = run_benchmark.build_arg_parser()
        parsed = parser.parse_args(["--bucket_name=my-test-bucket"])
        expected_flags = {
            "project_id",
            "bucket_name",
            "zone",
            "cluster_name",
            "network_name",
            "subnet_name",
            "machine_type",
            "node_pool_name",
            "gcsfuse_branch",
            "reservation_name",
            "no_cleanup",
            "iterations",
            "performance_threshold_gbps",
            "pod_timeout_seconds",
            "skip_csi_driver_build",
            "client_protocol",
        }
        self.assertEqual(set(vars(parsed).keys()), expected_flags)
        self.assertEqual(parsed.bucket_name, "my-test-bucket")
        self.assertEqual(parsed.cluster_name, "gke-pygrain-benchmark-cluster")
        self.assertEqual(parsed.performance_threshold_gbps, 13.0)
        self.assertEqual(parsed.iterations, 20)
        self.assertEqual(parsed.client_protocol, "http1")

        # Verify --help exits 0 and exposes only the 16 standard flags
        res = subprocess.run(
            [
                sys.executable,
                os.path.join(SCRIPT_DIR, "run_benchmark.py"),
                "--help",
            ],
            capture_output=True,
            text=True,
            check=False,
        )
        self.assertEqual(res.returncode, 0)
        for flag in expected_flags:
            self.assertIn(f"--{flag}", res.stdout)
        self.assertNotIn("--from_pod_logs_file", res.stdout)


if __name__ == "__main__":
    unittest.main()
