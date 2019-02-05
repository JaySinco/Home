%{
package main

import (
	"fmt"
    "math/big"
)
%}

%union {
    num  *big.Float
    sym  *symEntry
    arg  []*big.Float
}

%token<num> CONSTANT 
%token<sym> IDENTIFIER

%type<arg>  argument_expression_list
%type<num>  primary_expression multiplicative_expression additive_expression expression unary_expression postfix_expression

%start statement_list

%%

primary_expression
	: IDENTIFIER            
    {
        $$ = $1.num
        if $$ == nil {
            yylex.Error(fmt.Sprintf("unknown variable %q", $1.id))
        }
    }
	| CONSTANT             
	| '(' expression ')'    {$$ = $2}
	;

postfix_expression
    : primary_expression
    | IDENTIFIER '(' argument_expression_list ')'   
    {
        var err error
        $$, err = $1.fn($3...)
        if err != nil {
            yylex.Error(fmt.Sprintf("call func %q: %s", $1.id, err))
        }
    }
    ;

unary_expression
    : postfix_expression
    | '-' postfix_expression  {$$ = wrap1($2, (*big.Float).Neg)}
    ;

multiplicative_expression
	: unary_expression
	| multiplicative_expression '*' primary_expression   {$$ = wrap2($1, $3, (*big.Float).Mul)}
	| multiplicative_expression '/' primary_expression   {$$ = wrap2($1, $3, (*big.Float).Quo)}
	;

additive_expression
	: multiplicative_expression
	| additive_expression '+' multiplicative_expression  {$$ = wrap2($1, $3, (*big.Float).Add)}
	| additive_expression '-' multiplicative_expression  {$$ = wrap2($1, $3, (*big.Float).Sub)}
	;

expression
	: additive_expression
	;

argument_expression_list
	: expression  {$$ = make([]*big.Float, 0); $$ = append($$, $1)}
	| argument_expression_list ',' expression {$1 = append($1, $3); $$ = $1}
	;

expression_statement
	: ';'
	| expression ';'  {fmt.Printf("%s\n", format($1))}
	;

assignment_statement
    : IDENTIFIER '=' expression ';'  {$1.num = $3}

statement
	: expression_statement
    | assignment_statement
	;

statement_list
	: statement
	| statement_list statement
	;

%%

