using System.IO;
using System.Linq;
using System.Xml.Linq;
using WixSharp;
using File = System.IO.File;

namespace WixSetup
{
    public class CompressedDir<TCompressionStrategy> : WixSharp.File where TCompressionStrategy : ICompressionStrategy, new()
    {
        private readonly string _sourceDir;

        public CompressedDir(IWixProjectEvents wixProjectEvents, string targetPath, string sourceDir)
            : base($"{targetPath}.COMPRESSED")
        {
            _sourceDir = sourceDir;
            wixProjectEvents.WixSourceGenerated += OnWixSourceGenerated;
        }

        public void OnWixSourceGenerated(XDocument document)
        {
            var compressionStrategy = new TCompressionStrategy();
            var filesInSourceDir = new DirectoryInfo(_sourceDir)
                .EnumerateFiles("*", SearchOption.AllDirectories)
                .ToArray();
            var sourceDirName = Path.GetFileName(_sourceDir);
            var directorySize = filesInSourceDir
                .Sum(file => file.Length)
                .ToString();
            document
                .Select("Wix/Product")
                .AddElement("Property", $"Id={sourceDirName.ToUpper()}_SIZE; Value={directorySize}");

#if DEBUG
            // In debug mode, skip generating the file if it
            // already exists. Delete the file to regenerate it.
            if (File.Exists(Name))
            {
                return;
            }
#endif
            var tar = $"{Name}.tar";

            compressionStrategy.Compress(filesInSourceDir, Name, _sourceDir);
        }
    }
}
