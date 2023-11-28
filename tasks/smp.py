import os
import shutil
from .agent import hacky_dev_image_build as hacky_dev_image_build_agent
from invoke import task

@task
def list_workload_experiments(ctx):
    path = os.path.join(os.getcwd(), "test/workload-checks/typical/cases")
    print(f"Workload experiments are located in {path}, options:")
    print("\n".join(os.listdir(path)))


@task
def list_regression_experiments(ctx):
    path = os.path.join(os.getcwd(), "test/regression/cases")
    print(f"Regression experiments are located in {path}, options:")
    print("\n".join(os.listdir(path)))

def check_arch_and_

@task
def hacky_local_run(ctx, workload_experiment = None, regression_experiment = None, extra_smp_args = "", skip_build = False):
    if workload_experiment == None and regression_experiment == None:
        print("No experiment specified, please specify either --workload-experiment or --regression-experiment")
        return
    experiment = regression_experiment
    if workload_experiment != None:
        experiment = workload_experiment

    # `smp` and `lading` binaries must be on PATH
    # Future improvement would be to auto-fetch these
    smp_bin_path = shutil.which("smp")
    if smp_bin_path == None:
        print("'smp' binary not found on path. Install via `cargo install --git https://github.com/DataDog/single-machine-performance smp --bin smp`")
        return
    lading_bin_path = shutil.which("lading")
    if lading_bin_path == None:
        print("'lading' binary not found on path. Install via `cargo install --git https://github.com/DataDog/lading lading --bin lading`")
        return
    # TODO also check that both binaries match the current arch (current arch == hacky-dev-image-build arch)

    if not skip_build:
        hacky_dev_image_build_agent(ctx, target_image="smp-local-agent")

    experiment_dir = os.path.join(os.getcwd(), "test")
    if workload_experiment != None:
        experiment_dir = os.path.join(experiment_dir, "workload-checks/typical")
    if regression_experiment != None:
        experiment_dir = os.path.join(experiment_dir, "regression")

    ctx.run(f'{smp_bin_path} local-run --experiment-dir {experiment_dir} --case {experiment} --target-image smp-local-agent:latest --lading-path {lading_bin_path} --target-command "/bin/entrypoint.sh" --target datadog-agent {extra_smp_args}')
