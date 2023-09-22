name "pip3"

# The version of pip used must be at least equal to the one bundled with the Python version we use
# Python 3.11.5 bundles pip 23.2.1
default_version "23.2.1"

skip_transitive_dependency_licensing true

dependency "python3"

source :url => "https://github.com/pypa/pip/archive/#{version}.tar.gz",
       :sha256 => "975e6b09fe9d14927b67db05d7de3a60503a1696c8c23ca2486f114c20097ad4",
       :extract => :seven_zip

relative_path "pip-#{version}"

build do
  license "MIT"
  license_file "https://raw.githubusercontent.com/pypa/pip/main/LICENSE.txt"

  if ohai["platform"] == "windows"
    python = "#{windows_safe_path(python_3_embedded)}\\python.exe"
  else
    python = "#{install_dir}/embedded/bin/python3"
  end

  command "#{python} -m pip install ."

  if ohai["platform"] != "windows"
    block do
      FileUtils.rm_f(Dir.glob("#{install_dir}/embedded/lib/python3.*/site-packages/pip-*-py3.*.egg/pip/_vendor/distlib/*.exe"))
      FileUtils.rm_f(Dir.glob("#{install_dir}/embedded/lib/python3.*/site-packages/pip/_vendor/distlib/*.exe"))
    end
  end
end
