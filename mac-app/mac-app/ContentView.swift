//
//  ContentView.swift
//  mac-app
//
//  Created by rachelie on 2/25/25.
//

import SwiftUI

enum InputFileType: String, CaseIterable, Identifiable {
    case markdown, typst
    var id: Self { self }
}

enum OutputFileType: String, CaseIterable, Identifiable {
    case pdf, typst
    var id: Self { self }
}

enum TemplateOption: String, CaseIterable, Identifiable {
    case assignment, custom, cv, report
    var id: Self { self }
}
struct ContentView: View {
    @State private var filename: String = ""
    @State private var inputFileType: InputFileType = .markdown
    @State private var outputFileType: OutputFileType = .pdf
    @State private var password: String = ""
    @State private var repeatPassword: String = ""
    @State private var deleteOriginalFile: Bool = false
    @State private var selectedTemplate: TemplateOption = .assignment
    
    
    let templateOptions = ["CV", "Report", "Assignment", "Custom"]
    
    var body: some View {
        VStack(spacing: 0) {
            VStack (alignment: .leading, spacing: UIConstants.spacing){
                Text("Conversion Settings").font(.title2).bold()
                
                HStack {
                    Text("Output File Name:")
                    TextField("defaults to input file name", text: /*@START_MENU_TOKEN@*//*@PLACEHOLDER=Value@*/.constant("")/*@END_MENU_TOKEN@*/)
                }.frame(width:280)
                
                HStack {
                    Picker("Select Template: ", selection: $selectedTemplate) {
                        ForEach(TemplateOption.allCases) {
                            t in Text(t.rawValue.lowercased())
                        }
                    }
                    .pickerStyle(MenuPickerStyle())
                    .frame(width: 280)
                }
                
                HStack{
                    Picker("Input File Type: ", selection: $inputFileType){
                        ForEach(InputFileType.allCases) {
                            f in Text(f.rawValue.lowercased())
                        }
                    }.pickerStyle(MenuPickerStyle())
                        .frame(width: 280)
                }
                
                HStack {
                    Picker("Output File Type: ", selection: $outputFileType) {
                        ForEach(OutputFileType.allCases) {
                            f in Text(f.rawValue.lowercased())
                        }
                    }.pickerStyle(MenuPickerStyle())
                        .frame(width: 280)
                }
                
                HStack {
                    Text("Password:")
                    SecureField("", text: $password)
                        .textFieldStyle(RoundedBorderTextFieldStyle())
                        .frame(width: 150)
                    Button(action: { password = "" }) {
                        Image(systemName: "eye.slash")
                    }
                }
                
                HStack {
                    Text("Repeat:").frame(alignment: .leading)
                    SecureField("", text: $repeatPassword)
                        .textFieldStyle(RoundedBorderTextFieldStyle())
                }.frame(width: 280)
                
                VStack(alignment: .leading){
                    Toggle("Delete Original File", isOn: $deleteOriginalFile)
                }

            }
//            .background(Color(NSColor.windowBackgroundColor))
        }.frame(width: UIConstants.windowWidth, height: UIConstants.settingsHeight).padding(.top, 15)
        
        Divider().frame(width: UIConstants.windowWidth)
        
        HStack(alignment: .center, spacing: UIConstants.spacing){
            Button("Browse Files or Drag & Drop") {
                /*@START_MENU_TOKEN@*//*@PLACEHOLDER=Action@*/ /*@END_MENU_TOKEN@*/
            }
        }.frame(width: UIConstants.windowWidth, height: UIConstants.buttonHeight)
            .padding(UIConstants.padding).padding(.bottom, 10)
    }
}

#Preview {
    ContentView()
}
