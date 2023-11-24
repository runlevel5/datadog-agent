using Microsoft.Deployment.WindowsInstaller;
using System;
using System.IO;
using Datadog.CustomActions.Extensions;
using System.Text;
using Datadog.CustomActions.Interfaces;
using ICSharpCode.SharpZipLib.Tar;

namespace Datadog.CustomActions
{
    public class PythonDistributionCustomAction
    {
        static void Decompress(
            ISession session,
            string compressedFileName,
            string pythonDistributionName,
            int pythonDistributionSize)
        {
            using var progressRecordObj = new Record(2);

            progressRecordObj.SetInteger(1, 2);

            using var actionRecord = new Record(
                "Decompress Python distribution",
                $"Decompressing {pythonDistributionName} distribution",
                "[1] of [2]"
            );
            session.Message(InstallMessage.ActionStart, actionRecord);

            actionRecord.SetInteger(2, pythonDistributionSize);

            var decoder = new SevenZip.Compression.LZMA.Decoder();
            Int64 total = 0;
            using (var inStream = File.OpenRead(compressedFileName))
            {
                using (var outStream = File.Create($"{compressedFileName}.tar"))
                {
                    var reader = new BinaryReader(inStream, Encoding.UTF8);
                    // Properties of the stream are encoded on 5 bytes
                    var props = reader.ReadBytes(5);
                    decoder.SetDecoderProperties(props);
                    var length = reader.ReadInt64();

                    decoder.Code(inStream, outStream, inStream.Length, length, null);
                    outStream.Flush();
                    total += length;

                    actionRecord.SetInteger(1, (int)total);
                    session.Message(InstallMessage.ActionData, actionRecord);

                    progressRecordObj.SetInteger(2, (int)length);
                    session.Message(InstallMessage.Progress, progressRecordObj);
                }
            }
            var outputPath = Path.GetDirectoryName(Path.GetFullPath(compressedFileName));
            using (var inStream = File.OpenRead($"{compressedFileName}.tar"))
            using (var tarArchive = TarArchive.CreateInputTarArchive(inStream, Encoding.UTF8))
            {
                tarArchive.ExtractContents(outputPath);
            }
            File.Delete($"{compressedFileName}.tar");
            File.Delete($"{compressedFileName}");
        }

        private static ActionResult DecompressPythonDistribution(
            ISession session,
            string outputDirectoryName,
            string compressedDistributionFile,
            string pythonDistributionName,
            int pythonDistributionSize)
        {
            var projectLocation = session.Property("PROJECTLOCATION");

            try
            {
                var embedded = Path.Combine(projectLocation, compressedDistributionFile);
                var outputPath = Path.Combine(projectLocation, outputDirectoryName);

                // ensure extract result directory is empty so that we don't merge the directories
                // for different installs. The uninstaller should have already removed/backed up its
                // embedded directories, so this is just in case the uninstaller failed to do so.
                if (Directory.Exists(outputPath))
                {
                    session.Log($"Deleting directory \"{outputPath}\"");
                    Directory.Delete(outputPath, true);
                }
                else
                {
                    session.Log($"{outputPath} not found, skip deletion.");
                }

                if (File.Exists(embedded))
                {
                    // ensure extract result directory is empty so that we don't merge the directories
                    // for different installs. The uninstaller should have already removed/backed up its
                    // embedded directories, so this is just in case the uninstaller failed to do so.
                    if (Directory.Exists(outputPath))
                    {
                        session.Log($"Deleting directory \"{outputPath}\"");
                        Directory.Delete(outputPath, true);
                    }
                    else
                    {
                        session.Log($"{outputPath} not found, skip deletion.");
                    }

                    Decompress(session, embedded, pythonDistributionName, pythonDistributionSize);
                }
                else
                {
                    if (embedded.Contains("embedded3"))
                    {
                        throw new InvalidOperationException($"The file {embedded} doesn't exist, but it should");
                    }
                    session.Log($"{embedded} not found, skipping decompression.");
                }
            }
            catch (Exception e)
            {
                session.Log($"Error while decompressing {compressedDistributionFile}: {e}");
                return ActionResult.Failure;
            }

            return ActionResult.Success;
        }

        private static ActionResult DecompressPythonDistributions(ISession session)
        {
            var size = 0;
            var embedded2Size = session.Property("EMBEDDED2_SIZE");
            if (!string.IsNullOrEmpty(embedded2Size))
            {
                size = int.Parse(embedded2Size);
            }
            var actionResult = DecompressPythonDistribution(session, "embedded2", "embedded2.COMPRESSED", "Python 2", size);
            if (actionResult != ActionResult.Success)
            {
                return actionResult;
            }
            var embedded3Size = session.Property("EMBEDDED3_SIZE");
            if (!string.IsNullOrEmpty(embedded3Size))
            {
                size = int.Parse(embedded3Size);
            }
            return DecompressPythonDistribution(session, "embedded3", "embedded3.COMPRESSED", "Python 3", size);
        }

        [CustomAction]
        public static ActionResult DecompressPythonDistributions(Session session)
        {
            if (session.GetMode(InstallRunMode.Scheduled))
            {
                return DecompressPythonDistributions(new SessionWrapper(session));
            }
            return PrepareDecompressPythonDistributions(new SessionWrapper(session));
        }

        private static ActionResult PrepareDecompressPythonDistributions(ISession session)
        {
            try
            {
                var total = 0;
                var embedded2Size = session.Property("EMBEDDED2_SIZE");
                if (!string.IsNullOrEmpty(embedded2Size))
                {
                    total += int.Parse(embedded2Size);
                }
                var embedded3Size = session.Property("EMBEDDED3_SIZE");
                if (!string.IsNullOrEmpty(embedded3Size))
                {
                    total += int.Parse(embedded3Size);
                }
#if false
                // Add embedded Python size to the progress bar size
                // Even though we can't record accurate progress, it will look like it's
                // moving every time we decompress a Python distribution.
                using var record = new Record(MessageRecordFields.ProgressAddition, total);
                if (session.Message(InstallMessage.Progress, record) != MessageResult.OK)
                {
                    session.Log("Could not set the progress bar size");
                    return ActionResult.Failure;
                }
#else
                using var progressRecordObj = new Record(2);
                progressRecordObj.SetInteger(1, 3);
                progressRecordObj.SetInteger(2, total);
                if (session.Message(InstallMessage.Progress, progressRecordObj) != MessageResult.OK)
                {
                    session.Log("Could not set the progress bar size");
                    return ActionResult.Failure;
                }
                else
                {
                    session.Log($"Added {total} to the progress bar.");
                }
#endif
            }
            catch (Exception e)
            {
                session.Log($"Error settings the progress bar size: {e}");
                return ActionResult.Failure;
            }

            return ActionResult.Success;
        }
    }
}
