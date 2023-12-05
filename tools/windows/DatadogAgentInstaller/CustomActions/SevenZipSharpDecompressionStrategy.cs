using System.IO;
using System.Reflection;
using Datadog.CustomActions.Interfaces;
using SevenZip;

namespace Datadog.CustomActions
{
    public class SevenZipSharpDecompressionStrategy : IDecompressionStrategy
    {
        public const string SevenZipDllId = "SevenZipDll";

        public void Decompress(ISession session, string compressedFileName)
        {
            var assembly = Assembly.GetExecutingAssembly();
            var outputFile = Path.Combine(Path.GetDirectoryName(assembly.Location), "7z.dll");
            using (var resource = assembly.GetManifestResourceStream("Datadog.CustomActions.7z.dll"))
            {
                using (var file = new FileStream(outputFile, FileMode.Create, FileAccess.Write))
                {
                    resource.CopyTo(file);
                }
            }

            SevenZipBase.SetLibraryPath(outputFile);

            var outputPath = Path.GetDirectoryName(Path.GetFullPath(compressedFileName));
            using (var extractor = new SevenZipExtractor(compressedFileName))
            {
                extractor.ExtractArchive(outputPath);
            }
            File.Delete($"{compressedFileName}");
        }
    }
}
