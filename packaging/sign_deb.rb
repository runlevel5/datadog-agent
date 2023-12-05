#!/usr/bin/env ruby

require 'open3'
require 'tmpdir'

Result = Struct.new(:stdout, :stderr, :status)

def shellout(cmd, stdin: nil)
  stdout, stderr, status = Open3.capture3(cmd, stdin_data: stdin)
  Result.new(stdout, stderr, status)
end

def shellout!(cmd, stdin: nil)
  result = shellout(cmd, stdin: stdin)
  raise "cmd failed: #{result.status}" unless result.status.success?

  result.stdout
end

def require_env(name)
  val = ENV[name]
  raise "#{name} unset but required" if val.nil? || val.empty?

  val
end

deb_file = File.expand_path(require_env('DEB_FILE'))
deb_gpg_key_name = require_env('DEB_GPG_KEY_NAME')
deb_gpg_key_ssm_name = require_env('DEB_GPG_KEY_SSM_NAME')
deb_gpg_signing_passphrase_ssm_name = require_env('DEB_SIGNING_PASSPHRASE_SSM_NAME')

gpg = if shellout('which gpg2').status.success?
        'gpg2'
      elsif shellout('which gpg').status.success?
        'gpg'
      else
        raise 'no gpg found'
      end

gpg_key = shellout!("aws ssm get-parameter --region us-east-1 --name #{deb_gpg_key_ssm_name} --with-decryption --query \"Parameter.Value\" --out text")
shellout!('gpg --import --batch', stdin: gpg_key)

signing_passphrase = shellout!("aws ssm get-parameter --region us-east-1 --name #{deb_gpg_signing_passphrase_ssm_name} --with-decryption --query \"Parameter.Value\" --out text")

Dir.mktmpdir do |tmp|
  Dir.chdir(tmp) do
    # Extract the deb file contents
    shellout!("ar x #{deb_file}")

    # Concatenate contents, in order per +debsigs+ documentation.
    shellout!('cat debian-binary control.tar.* data.tar.* > complete')

    # Create signature (as +root+)
    gpg_command =  "#{gpg} --armor --sign --detach-sign"
    gpg_command << " --local-user '#{deb_gpg_key_name}'"
    gpg_command << " --homedir #{ENV['HOME']}/.gnupg"

    ## pass the +signing_passphrase+ via +STDIN+
    gpg_command << ' --batch --no-tty'

    ## Check `gpg` for the compatibility/need of pinentry-mode
    # - We're calling gpg with the +--pinentry-mode+ argument, and +STDIN+ of +/dev/null+
    # - This _will_ fail with exit code 2 no matter what. We want to check the +STDERR+
    #   for the error message about the parameter. If it is _not present_ in the
    #   output, then we _do_ want to add it. (If +grep -q+ is +1+, add parameter)
    unless shellout("#{gpg} --pinentry-mode loopback </dev/null 2>&1 | grep -q pinentry-mode").status.success?
      gpg_command << ' --pinentry-mode loopback'
    end
    gpg_command << ' --passphrase-fd 0'
    gpg_command << ' -o _gpgorigin complete'

    shellout!("fakeroot #{gpg_command}", stdin: signing_passphrase)

    # Append +_gpgorigin+ to the +.deb+ file (as +root+)
    shellout!("fakeroot ar rc #{deb_file} _gpgorigin")
  end
end
