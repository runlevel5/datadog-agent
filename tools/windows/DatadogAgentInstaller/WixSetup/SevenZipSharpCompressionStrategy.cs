using System;
using System.IO;

namespace WixSetup
{
    public class SevenZipSharpCompressionStrategy : ICompressionStrategy
    {
        public void Compress(FileInfo[] files, string archiveName, string sourceDir)
        {
            using var pProcess = new System.Diagnostics.Process();
            pProcess.StartInfo.FileName = "7z.exe";
            pProcess.StartInfo.Arguments = $"a -mx=5 -ms=on {archiveName} {sourceDir}";
            pProcess.StartInfo.UseShellExecute = false;
            pProcess.StartInfo.RedirectStandardOutput = true;
            pProcess.StartInfo.WindowStyle = System.Diagnostics.ProcessWindowStyle.Hidden;
            pProcess.StartInfo.CreateNoWindow = true;
            pProcess.Start();
            var output = pProcess.StandardOutput.ReadToEnd();
            Console.WriteLine(output);
            pProcess.WaitForExit();
        }
    }
}
