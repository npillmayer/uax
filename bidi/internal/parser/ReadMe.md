This package is not intended for public use and may not be imported.

The package contains a draft implementation which explored the feasiblity of an Eerley-parser for
UAX#9. The rules of UAX#9 are context sensitive grammar rules, while Earley parsing is used for 
context free grammars. However, GoRGOs Earley parser is very lenient with ambiguous grammars and I
explored the possibility of rewriting the UAX#9 rules in a context free grammar and LR-parse it.
The results were interesting and greatly improved the robustness of GoRGOs Earley parser with
highly ambiguous grammars, but in the end I abandoned the experiment.

I'll keep the code around for later reference and for having a playing-field for highly ambiguous grammars.

Peace!
