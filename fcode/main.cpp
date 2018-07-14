#include <vector>
#include <string>
#include <regex>
#include <iostream>
#include <fstream>
#include <sstream>
#include <iomanip>
#include <cctype>
#include <algorithm> 

#include <boost/filesystem.hpp> 


int calc_line_num(std::string &filename)
{
	int lines_count = 0;
	std::string line_content;
	std::ifstream infile(filename);
	while (std::getline(infile, line_content))
		lines_count++;
	return lines_count;
}

std::string &ltrim(std::string &ss)
{
	auto p = find_if(ss.begin(), ss.end(), [](int c) { return !std::isspace(c); });
	ss.erase(ss.begin(), p);
	return ss;
}

int main()
{
	std::cout << "Please enter filename filter(default '.'): ";
	std::string regexp_filter;
	std::getline(std::cin, regexp_filter);
	if (regexp_filter.size() == 0)
		regexp_filter = ".";
	std::regex pattern(regexp_filter, std::regex::icase);

	std::vector<std::string> matched_filename_list;
	auto pwd = boost::filesystem::current_path();
	for (boost::filesystem::recursive_directory_iterator beg_iter(pwd), end_iter;
		beg_iter != end_iter; ++beg_iter)
	{
		std::string str_path = beg_iter->path().string();
		if(std::regex_search(str_path, pattern))
			matched_filename_list.push_back(str_path);
	}

	std::cout << "\n\n      Lines |  Filepath" << std::endl;
	std::cout << "      -----------------------------------------------------" << std::endl;
	int sum_lines = 0;
	for (auto filename: matched_filename_list) {
		int nline = calc_line_num(filename);
		filename.replace(0, pwd.size(), ".");
		std::cout << std::setw(10) << nline << "  | " << filename << std::endl;
		sum_lines += nline;
	}
	std::cout << "      -----------------------------------------------------" << std::endl;
	std::cout << std::setw(10) << sum_lines << "  | Total\n\n\n" << std::endl;

	std::string keyword;
	for (;;) {
		std::cout << "Enter keyword [quit] to exit :  ";
		std::getline(std::cin, keyword);
		std::cout << "\n" << std::endl;
		if (keyword == "quit")
			break;
		std::regex search_key(keyword, std::regex::icase);
		for (auto filename : matched_filename_list) 
		{
			std::ifstream infile(filename);
			std::string line;
			std::stringstream buf;
			const int max_word = 60;
			bool should_print = false;
			int lineno = 0;
			while (std::getline(infile, line))
			{
				lineno++;
				ltrim(line);
				if (std::regex_search(line, search_key))
				{
					should_print = true;
					buf << std::setw(4) << lineno << "  ->  " << std::setw(max_word) 
						<< std::left << line << std::right;
					if (line.size() > max_word)
						buf << "...";
					buf << "\n";
				}
			}
			if (should_print)
			{
				filename.replace(0, pwd.size(), ".");
				std::cout << "***** " << filename << " *****\n\n";
				std::cout << buf.str() << "\n" << std::endl;
			}
		}
	}

	return 0;
}
